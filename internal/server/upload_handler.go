package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const maxUploadSize = 5 << 20 // 5 MB

// allowedMimeTypes maps MIME type → file extension.
var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

func (s *Server) handleUploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "File too large (max 5MB)", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Missing file field", nil)
		return
	}
	defer file.Close()

	// Validate content type.
	ct := header.Header.Get("Content-Type")
	ext, ok := allowedMimeTypes[ct]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Unsupported file type. Allowed: JPEG, PNG, GIF, WebP", nil)
		return
	}

	// Generate unique filename.
	filename := uuid.New().String() + ext

	// Ensure upload directory exists.
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		s.logger.Error("upload: mkdir failed", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Sanitize and create destination file.
	destPath := filepath.Join(s.uploadDir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		s.logger.Error("upload: create file failed", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		s.logger.Error("upload: write file failed", "error", err)
		os.Remove(destPath)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	url := fmt.Sprintf("/api/v1/images/%s", filename)
	s.writeJSON(w, http.StatusCreated, map[string]string{
		"url":      url,
		"filename": filename,
	})
}

func (s *Server) handleServeImage(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")

	// Sanitize: only allow simple filenames (UUID + ext), no path traversal.
	if strings.ContainsAny(filename, "/\\..") || filename == "" {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid filename", nil)
		return
	}

	filePath := filepath.Join(s.uploadDir, filename)
	http.ServeFile(w, r, filePath)
}

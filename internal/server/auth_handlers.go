package server

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/like19970403/TRPG-Simulation/internal/auth"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateRegister(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	hash, err := auth.HashPassword(req.Password, s.bcryptCost)
	if err != nil {
		s.logger.Error("auth: hash password", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	user, err := s.authRepo.CreateUser(r.Context(), req.Username, req.Email, hash)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			s.writeError(w, http.StatusConflict, "CONFLICT", "Username or email already exists", nil)
			return
		}
		s.logger.Error("auth: create user", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, RegisterResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validateLogin(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	user, err := s.authRepo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil)
		return
	}

	if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
		s.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil)
		return
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username, s.jwtSecret, s.accessTTL)
	if err != nil {
		s.logger.Error("auth: generate access token", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	raw, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		s.logger.Error("auth: generate refresh token", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	expiresAt := time.Now().Add(s.refreshTTL)
	if err := s.authRepo.StoreRefreshToken(r.Context(), user.ID, hash, expiresAt); err != nil {
		s.logger.Error("auth: store refresh token", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.setRefreshTokenCookie(w, raw, s.refreshTTL)
	s.writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.accessTTL.Seconds()),
		TokenType:   "Bearer",
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing refresh token", nil)
		return
	}

	tokenHash := auth.HashRefreshToken(cookie.Value)
	rt, err := s.authRepo.GetRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid refresh token", nil)
		return
	}

	// Token theft detection: revoked token reuse
	if rt.Revoked {
		s.authRepo.RevokeAllUserRefreshTokens(r.Context(), rt.UserID)
		s.clearRefreshTokenCookie(w)
		s.writeError(w, http.StatusUnauthorized, "TOKEN_REVOKED", "Refresh token has been revoked", nil)
		return
	}

	if time.Now().After(rt.ExpiresAt) {
		s.writeError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Refresh token has expired", nil)
		return
	}

	// Revoke old token (rotation)
	if err := s.authRepo.RevokeRefreshToken(r.Context(), rt.ID); err != nil {
		s.logger.Error("auth: revoke old refresh token", "error", err)
	}

	// Look up user for access token claims
	user, err := s.authRepo.GetUserByID(r.Context(), rt.UserID)
	if err != nil {
		s.logger.Error("auth: get user for refresh", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username, s.jwtSecret, s.accessTTL)
	if err != nil {
		s.logger.Error("auth: generate access token", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	raw, hash, err := auth.GenerateRefreshToken()
	if err != nil {
		s.logger.Error("auth: generate refresh token", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	expiresAt := time.Now().Add(s.refreshTTL)
	if err := s.authRepo.StoreRefreshToken(r.Context(), user.ID, hash, expiresAt); err != nil {
		s.logger.Error("auth: store refresh token", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	s.setRefreshTokenCookie(w, raw, s.refreshTTL)
	s.writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.accessTTL.Seconds()),
		TokenType:   "Bearer",
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		tokenHash := auth.HashRefreshToken(cookie.Value)
		rt, err := s.authRepo.GetRefreshTokenByHash(r.Context(), tokenHash)
		if err == nil {
			if err := s.authRepo.RevokeRefreshToken(r.Context(), rt.ID); err != nil {
				s.logger.Error("auth: revoke refresh token on logout", "error", err)
			}
		}
	}

	s.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	claims := UserClaimsFromContext(r.Context())

	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if errs := validatePasswordChange(req); len(errs) > 0 {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", errs)
		return
	}

	user, err := s.authRepo.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		s.logger.Error("auth: get user for password change", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if err := auth.CheckPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		s.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Current password is incorrect", nil)
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword, s.bcryptCost)
	if err != nil {
		s.logger.Error("auth: hash new password", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	if err := s.authRepo.UpdatePassword(r.Context(), claims.UserID, newHash); err != nil {
		s.logger.Error("auth: update password", "error", err)
		s.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
		return
	}

	// Revoke all refresh tokens (invalidate all sessions)
	if err := s.authRepo.RevokeAllUserRefreshTokens(r.Context(), claims.UserID); err != nil {
		s.logger.Error("auth: revoke tokens after password change", "error", err)
	}

	s.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// --- helpers ---

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("failed to encode response", "error", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string, details []ErrorDetail) {
	s.writeJSON(w, status, ErrorResponse{
		Error:   code,
		Message: message,
		Details: details,
	})
}

func (s *Server) setRefreshTokenCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func (s *Server) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func validateRegister(req RegisterRequest) []ErrorDetail {
	var errs []ErrorDetail
	if len(req.Username) < 3 || len(req.Username) > 50 {
		errs = append(errs, ErrorDetail{Field: "username", Reason: "must be between 3 and 50 characters"})
	} else if !usernameRegex.MatchString(req.Username) {
		errs = append(errs, ErrorDetail{Field: "username", Reason: "must contain only alphanumeric characters and underscores"})
	}
	if _, err := mail.ParseAddress(req.Email); err != nil || req.Email == "" {
		errs = append(errs, ErrorDetail{Field: "email", Reason: "must be a valid email address"})
	} else if len(req.Email) > 255 {
		errs = append(errs, ErrorDetail{Field: "email", Reason: "must not exceed 255 characters"})
	}
	if len(req.Password) < 8 || len(req.Password) > 72 {
		errs = append(errs, ErrorDetail{Field: "password", Reason: "must be between 8 and 72 characters"})
	}
	return errs
}

func validateLogin(req LoginRequest) []ErrorDetail {
	var errs []ErrorDetail
	if _, err := mail.ParseAddress(req.Email); err != nil || req.Email == "" {
		errs = append(errs, ErrorDetail{Field: "email", Reason: "must be a valid email address"})
	}
	if req.Password == "" {
		errs = append(errs, ErrorDetail{Field: "password", Reason: "must not be empty"})
	}
	return errs
}

func validatePasswordChange(req PasswordChangeRequest) []ErrorDetail {
	var errs []ErrorDetail
	if req.CurrentPassword == "" {
		errs = append(errs, ErrorDetail{Field: "currentPassword", Reason: "must not be empty"})
	}
	if len(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		errs = append(errs, ErrorDetail{Field: "newPassword", Reason: "must be between 8 and 72 characters"})
	}
	return errs
}

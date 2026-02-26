package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/like19970403/TRPG-Simulation/internal/auth"
)

const userClaimsKey contextKey = "user_claims"

// UserClaimsFromContext extracts JWT claims from the request context.
func UserClaimsFromContext(ctx context.Context) *auth.Claims {
	if claims, ok := ctx.Value(userClaimsKey).(*auth.Claims); ok {
		return claims
	}
	return nil
}

// requireAuth validates the Bearer token and injects claims into context.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "Missing or invalid access token",
			})
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ValidateAccessToken(tokenString, s.jwtSecret)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error:   "UNAUTHORIZED",
				Message: "Missing or invalid access token",
			})
			return
		}

		ctx := context.WithValue(r.Context(), userClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

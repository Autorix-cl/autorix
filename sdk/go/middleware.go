package autorix

import (
	"net/http"
	"strings"
)

// Middleware extracts identity headers injected by Autorix Aegis (X-User-ID, X-User-Email, X-User-Roles)
func (c *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID != "" {
			email := r.Header.Get("X-User-Email")
			rawRoles := r.Header.Get("X-User-Roles")

			var roles []string
			if rawRoles != "" {
				roles = strings.Split(rawRoles, ",")
			}

			user := &User{
				ID:        userID,
				Email:     email,
				Roles:     roles,
				IsMachine: strings.HasPrefix(userID, "service_") || strings.HasPrefix(userID, "av_"),
			}

			ctx := WithUser(r.Context(), user)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth enforces that a request must have an authenticated identity
func (c *Client) RequireAuth(next http.Handler) http.Handler {
	return c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := UserFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthorized","message":"Missing authenticated identity"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

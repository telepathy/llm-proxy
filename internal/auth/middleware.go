package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/llm-proxy/internal/config"
)

// Middleware creates an HTTP basic authentication middleware
type Middleware struct {
	users   []config.UserConfig
	enabled bool
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(cfg config.AuthConfig) *Middleware {
	return &Middleware{
		users:   cfg.Users,
		enabled: cfg.Enabled,
	}
}

// Wrap wraps an http.Handler with authentication
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Extract credentials from Authorization header
		username, password, ok := m.extractCredentials(r)
		if !ok {
			m.unauthorized(w)
			return
		}

		// Validate credentials
		if !m.validateCredentials(username, password) {
			m.unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) extractCredentials(r *http.Request) (string, string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", "", false
	}

	// Support Basic auth
	if strings.HasPrefix(auth, "Basic ") {
		decoded, err := base64.StdEncoding.DecodeString(auth[6:])
		if err != nil {
			return "", "", false
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return "", "", false
		}
		return parts[0], parts[1], true
	}

	// Support Bearer auth (treat as password-only, username empty)
	if strings.HasPrefix(auth, "Bearer ") {
		return "", auth[7:], true
	}

	return "", "", false
}

func (m *Middleware) validateCredentials(username, password string) bool {
	for _, user := range m.users {
		// Use constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare(
			[]byte(username),
			[]byte(user.Username),
		) == 1

		// For passwords, compare hashed values
		passwordMatch := m.comparePassword(password, user.Password)

		if usernameMatch && passwordMatch {
			return true
		}
	}
	return false
}

func (m *Middleware) comparePassword(provided, stored string) bool {
	// If stored password starts with $2a$, $2b$, or $2y$, it's bcrypt
	if strings.HasPrefix(stored, "$2") {
		// For simplicity, we'll do direct comparison
		// In production, use golang.org/x/crypto/bcrypt
		return subtle.ConstantTimeCompare(
			[]byte(provided),
			[]byte(stored),
		) == 1
	}

	// Otherwise, use SHA-256 hash comparison
	providedHash := sha256.Sum256([]byte(provided))
	storedHash := sha256.Sum256([]byte(stored))

	return subtle.ConstantTimeCompare(providedHash[:], storedHash[:]) == 1
}

func (m *Middleware) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="LLM Proxy"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

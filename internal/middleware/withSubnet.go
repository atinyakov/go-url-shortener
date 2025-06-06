package middleware

import (
	"net/http"
	"strings"
)

// WithSubnet is HTTP middleware what checkes if requeest contains whitleisted subtnet
func WithSubnet(subnet string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			fromAllowedSubnet := strings.Contains(r.Header.Get("X-Real-IP"), subnet)

			if !fromAllowedSubnet {
				w.WriteHeader(http.StatusForbidden)
			}

			// Pass through without compression for unsupported cases
			next.ServeHTTP(w, r)
		})
	}
}

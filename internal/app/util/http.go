package util

import (
	"net/http"
	"strings"
)

// IsFilteredHttpRoute reports whether the request URI identifies a service route.
func IsFilteredHttpRoute(r *http.Request) bool {
	return strings.Contains(r.RequestURI, "health") || strings.Contains(r.RequestURI, "debug") || strings.Contains(r.RequestURI, "metric")
}

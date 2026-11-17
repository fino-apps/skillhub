package middleware

import (
	"net/http"

	"github.com/fino-io/gokit/middleware/accesslog"
)

func accessLogMiddleware() Middleware {
	// This Fino extension runs before the router, including before file payload decoding.
	return func(next http.Handler) http.Handler {
		return localRequestsOnly(accesslog.HTTPMiddleware()(next))
	}
}

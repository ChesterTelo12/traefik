package tracing

import (
	"context"
	"net"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/middlewares/forwardedheaders"
	"github.com/traefik/traefik/v3/pkg/tracing"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type tracingMiddleware struct {
	next http.Handler
	name string
}

func New(ctx context.Context, next http.Handler, name string) (http.Handler, error) {
	return &tracingMiddleware{
		next: next,
		name: name,
	}, nil
}

func (t *tracingMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if !tracing.IsEnabled(ctx) {
		t.next.ServeHTTP(w, req)
		return
	}

	ctx, span := tracing.StartSpan(ctx, req.Method+" "+req.URL.Path)
	defer span.End()

	// Call next handler first so forwarded headers middleware can resolve the client IP
	t.next.ServeHTTP(w, req.WithContext(ctx))

	// Retrieve the resolved client IP from context or fallback to RemoteAddr
	clientIP := req.RemoteAddr
	if val := ctx.Value(forwardedheaders.ClientIPCtxKey); val != nil {
		if ipStr, ok := val.(string); ok {
			clientIP = ipStr
		}
	}

	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}

	span.SetAttributes(semconv.ClientAddressKey.String(clientIP))
}

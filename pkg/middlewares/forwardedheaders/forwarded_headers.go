package forwardedheaders

import (
	"context"
	"net"
	"net/http"

	"github.com/traefik/traefik/v3/pkg/ip"
)

type contextKey string

const ClientIPCtxKey contextKey = "resolved-client-ip"

type forwardedHeaders struct {
	next           http.Handler
	trustedIPs     []*net.IPNet
	insecure       bool
	connectionIPs  *ip.Pool
}

func New(ctx context.Context, next http.Handler, trustedIPs []string, insecure bool) (http.Handler, error) {
	var ipNets []*net.IPNet
	for _, ipStr := range trustedIPs {
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return nil, err
		}
		ipNets = append(ipNets, ipNet)
	}

	return &forwardedHeaders{
		next:       next,
		trustedIPs: ipNets,
		insecure:   insecure,
	}, nil
}

func (f *forwardedHeaders) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	clientIP := req.RemoteAddr
	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		clientIP = host
	}

	if f.insecure {
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = xff
		}
	} else if len(f.trustedIPs) > 0 {
		// Check if remote IP is trusted
		remoteIP := net.ParseIP(clientIP)
		for _, ipNet := range f.trustedIPs {
			if ipNet.Contains(remoteIP) {
				if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
					clientIP = xff
				}
				break
			}
		}
	}

	req.RemoteAddr = clientIP
	ctx := context.WithValue(req.Context(), ClientIPCtxKey, clientIP)
	f.next.ServeHTTP(w, req.WithContext(ctx))
}

package jsonrpc

import (
	"context"
	"net/http"
)

type mdHeaderKey struct{}

// NewContextWithHeaders returns a copy of ctx with the given HTTP
// headers attached. When a Client makes a request using this context,
// the headers are merged into the request on top of any client-level
// headers set via WithHeader / WithHeaders. Headers set here take
// precedence over client-level headers but are overridden by the
// HTTPAuth callback.
//
// This is useful for per-call authentication or tracing headers
// without creating a separate Client per call.
func NewContextWithHeaders(ctx context.Context, h http.Header) context.Context {
	if len(h) == 0 {
		return ctx
	}
	var ctxh http.Header
	if prev, ok := ctx.Value(mdHeaderKey{}).(http.Header); ok {
		ctxh = setHeaders(prev.Clone(), h)
	} else {
		ctxh = h.Clone()
	}
	return context.WithValue(ctx, mdHeaderKey{}, ctxh)
}

func headersFromContext(ctx context.Context) http.Header {
	h, _ := ctx.Value(mdHeaderKey{}).(http.Header)
	return h
}

func setHeaders(dst, src http.Header) http.Header {
	for k, vs := range src {
		dst[http.CanonicalHeaderKey(k)] = vs
	}
	return dst
}

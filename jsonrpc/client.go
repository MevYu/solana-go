package jsonrpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// DefaultMaxIdleConnsPerHost is the per-host idle connection pool size used
// by NewClient when no WithMaxIdleConnsPerHost option is provided. Go's
// built-in default (http.DefaultMaxIdleConnsPerHost) is 2, which causes
// TCP+TLS re-dials as soon as a third goroutine calls the same endpoint
// concurrently. 10 is a conservative but practical default: it covers
// small applications and typical bot workloads without over-reserving
// file descriptors. Increase it with WithMaxIdleConnsPerHost for indexers
// or high-frequency trading systems (50–200 is typical).
const DefaultMaxIdleConnsPerHost = 10

// DefaultMaxIdleConns is the total idle connection pool size used by
// NewClient when no WithMaxIdleConns option is provided.
const DefaultMaxIdleConns = 100

// NewDefaultTransport returns an *http.Transport with Solana-optimised
// defaults. maxIdleConns is the global idle connection cap; maxIdleConnsPerHost
// sets the per-host keep-alive pool.
//
// Use this when you need to layer a custom RoundTripper (e.g. for metrics
// or mTLS) on top of the optimised transport rather than starting from
// http.DefaultTransport:
//
//	tr := rpc.NewDefaultTransport(rpc.DefaultMaxIdleConns, rpc.DefaultMaxIdleConnsPerHost)
//	tr.TLSClientConfig = &tls.Config{...}
//	c := rpc.NewClient(url, rpc.WithHTTPClient(&http.Client{Transport: tr}))
func NewDefaultTransport(maxIdleConns, maxIdleConnsPerHost int) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// HTTPAuth is a function called before each HTTP request to inject
// authentication headers. It is safe for concurrent use.
// Example usage: API key authentication for RPC providers.
type HTTPAuth func(h http.Header) error

// Client is a JSON-RPC 2.0 client for the Solana HTTP endpoint. It
// is safe for concurrent use by multiple goroutines.
//
// A zero Client is not usable; always construct one with NewClient.
type Client struct {
	endpoint            string
	httpClient          *http.Client
	headers             http.Header
	httpAuth            HTTPAuth
	codec               Codec
	retry               RetryPolicy
	maxIdleConns        int
	maxIdleConnsPerHost int
	nextID              atomic.Uint64
}

// ClientOption configures a Client at construction time.
// Both Config and the WithXxx helper functions implement this interface,
// so either style (or a mix) can be passed to NewClient.
type ClientOption interface {
	applyOption(*Client)
}

// clientOptionFunc lets a plain func(*Client) satisfy ClientOption.
type clientOptionFunc func(*Client)

func (f clientOptionFunc) applyOption(c *Client) { f(c) }

// Config is a struct-based alternative to the WithXxx functional options.
// Zero values are ignored; the client defaults are used for unset fields.
//
//	rpc.NewClient(url, rpc.Config{
//	    MaxIdleConns:        100,
//	    MaxIdleConnsPerHost: 20,
//	    Codec:               rpc.StdCodec(),
//	})
type Config struct {
	// HTTPClient replaces the default http.Client entirely.
	// When set, MaxIdleConns and MaxIdleConnsPerHost have no effect.
	HTTPClient *http.Client

	// MaxIdleConns is the total idle connection pool size across all hosts.
	// Default: DefaultMaxIdleConns (100).
	MaxIdleConns int

	// MaxIdleConnsPerHost is the per-host idle connection pool size.
	// Default: DefaultMaxIdleConnsPerHost (10).
	MaxIdleConnsPerHost int

	// Codec replaces the default GoJSONCodec.
	Codec Codec

	// RetryPolicy replaces the default exponential-backoff retry policy.
	RetryPolicy RetryPolicy

	// Headers are added to every request. Use for static auth headers.
	Headers http.Header

	// HTTPAuth is called before each request to inject auth headers dynamically.
	HTTPAuth HTTPAuth
}

func (cfg Config) applyOption(c *Client) {
	if cfg.HTTPClient != nil {
		c.httpClient = cfg.HTTPClient
	}
	if cfg.MaxIdleConns > 0 {
		c.maxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxIdleConnsPerHost > 0 {
		c.maxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
	}
	if cfg.Codec != nil {
		c.codec = cfg.Codec
	}
	if cfg.RetryPolicy != nil {
		c.retry = cfg.RetryPolicy
	}
	if len(cfg.Headers) > 0 {
		if c.headers == nil {
			c.headers = make(http.Header)
		}
		for k, vs := range cfg.Headers {
			c.headers[k] = vs
		}
	}
	if cfg.HTTPAuth != nil {
		c.httpAuth = cfg.HTTPAuth
	}
}

// WithHTTPClient replaces the default *http.Client. When set, the client
// is used as-is and MaxIdleConns/MaxIdleConnsPerHost have no effect. Callers
// who want to start from the library's transport defaults and only change one
// setting should use NewDefaultTransport instead:
//
//	tr := rpc.NewDefaultTransport(rpc.DefaultMaxIdleConns, 50)
//	c := rpc.NewClient(url, rpc.WithHTTPClient(&http.Client{Transport: tr}))
func WithHTTPClient(hc *http.Client) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	})
}

// WithMaxIdleConns sets the total number of idle (keep-alive) connections
// across all hosts for the default transport. The default is DefaultMaxIdleConns (100).
// Has no effect when WithHTTPClient is also passed.
func WithMaxIdleConns(n int) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if n > 0 {
			c.maxIdleConns = n
		}
	})
}

// WithMaxIdleConnsPerHost sets the per-host idle connection pool size for
// the default transport. The default is DefaultMaxIdleConnsPerHost (10).
// Has no effect when WithHTTPClient is also passed.
func WithMaxIdleConnsPerHost(n int) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if n > 0 {
			c.maxIdleConnsPerHost = n
		}
	})
}

// WithCodec replaces the default JSON codec. The default is
// GoJSONCodec; use WithCodec(StdCodec()) to opt out of the
// goccy/go-json dependency in favour of stdlib encoding/json.
func WithCodec(codec Codec) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if codec != nil {
			c.codec = codec
		}
	})
}

// WithRetryPolicy replaces the default retry policy.
func WithRetryPolicy(p RetryPolicy) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if p != nil {
			c.retry = p
		}
	})
}

// WithHeader sets a single HTTP header on every request.
func WithHeader(key, value string) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if c.headers == nil {
			c.headers = make(http.Header)
		}
		c.headers.Set(key, value)
	})
}

// WithHeaders sets multiple HTTP headers on every request.
func WithHeaders(headers http.Header) ClientOption {
	return clientOptionFunc(func(c *Client) {
		if c.headers == nil {
			c.headers = make(http.Header)
		}
		for k, vs := range headers {
			c.headers[k] = vs
		}
	})
}

// WithHTTPAuth sets a callback invoked before each request to inject
// authentication headers dynamically.
func WithHTTPAuth(a HTTPAuth) ClientOption {
	return clientOptionFunc(func(c *Client) {
		c.httpAuth = a
	})
}

// SetHeader adds or replaces a header on every future request. Use this
// for runtime header updates (e.g. rotating API keys) after the client
// has been created.
//
// SetHeader is NOT safe to call while requests are in flight: it mutates
// the shared header map that the request path concurrently reads. Set
// headers before issuing requests, or quiesce in-flight requests first.
func (c *Client) SetHeader(key, value string) {
	if c.headers == nil {
		c.headers = make(http.Header)
	}
	c.headers.Set(key, value)
}

// newClient is the shared construction core. It applies opts in order
// then builds the default transport if no custom HTTPClient was provided.
func newClient(endpoint string, opts []ClientOption) *Client {
	c := &Client{
		endpoint:            endpoint,
		codec:               GoJSONCodec(),
		retry:               DefaultRetryPolicy(),
		maxIdleConns:        DefaultMaxIdleConns,
		maxIdleConnsPerHost: DefaultMaxIdleConnsPerHost,
	}
	for _, opt := range opts {
		opt.applyOption(c)
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Transport: NewDefaultTransport(c.maxIdleConns, c.maxIdleConnsPerHost)}
	}
	return c
}

// NewClient constructs a Client from a Config struct.
// Zero-value fields in cfg fall back to library defaults.
//
//	c := rpc.NewClient("https://api.mainnet-beta.solana.com", rpc.Config{
//	    MaxIdleConns:        100,
//	    MaxIdleConnsPerHost: 20,
//	})
func NewClient(endpoint string, cfg Config) *Client {
	return newClient(endpoint, []ClientOption{cfg})
}

// NewClientWith constructs a Client using functional options.
//
//	c := rpc.NewClientWith("https://api.mainnet-beta.solana.com",
//	    rpc.WithMaxIdleConnsPerHost(20),
//	    rpc.WithHeader("X-API-Key", "secret"),
//	)
func NewClientWith(endpoint string, opts ...ClientOption) *Client {
	return newClient(endpoint, opts)
}

// Endpoint returns the endpoint URL the Client was constructed with.
func (c *Client) Endpoint() string { return c.endpoint }

// sendRaw runs the HTTP + retry loop against a build callback that produces
// the JSON request body, and returns the raw response body.
//
// build is invoked once per attempt; it must return a fresh []byte each
// time because the HTTP layer may consume the reader on transport failure.
// label is a short tag prepended to error messages ("getBalance",
// "batch[3]", …).
func (c *Client) sendRaw(ctx context.Context, label string, build func() ([]byte, error)) ([]byte, error) {
	// Lazy timer: created on the first retry and reused thereafter.
	// defer stops it on any exit path, preventing the goroutine leak
	// that time.After would cause on ctx cancellation.
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for attempt := 1; ; attempt++ {
		body, err := build()
		if err != nil {
			return nil, fmt.Errorf("solana rpc %s: marshal request: %w", label, err)
		}

		resp, err := c.doHTTP(ctx, body)
		if err == nil {
			return resp, nil
		}
		delay, retry := c.retry.ShouldRetry(attempt, err)
		if !retry {
			return nil, fmt.Errorf("solana rpc %s: %w", label, err)
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("solana rpc %s: %w", label, ctx.Err())
		case <-timer.C:
		}
	}
}

// callRaw is the single-request convenience on top of sendRaw used by
// CallContext.
func (c *Client) callRaw(ctx context.Context, method string, args []any) ([]byte, error) {
	if args == nil {
		args = []any{}
	}
	req := Request{
		Version: jsonrpcVersion,
		ID:      c.nextID.Add(1),
		Method:  method,
		Params:  args,
	}
	return c.sendRaw(ctx, method, func() ([]byte, error) {
		return c.codec.Marshal(&req)
	})
}

// doHTTP sends body as a JSON POST request and returns the response body.
// On a non-2xx status it returns an *httpError containing the response body.
func (c *Client) doHTTP(ctx context.Context, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	for k, vs := range c.headers {
		httpReq.Header[k] = vs
	}
	for k, vs := range headersFromContext(ctx) {
		httpReq.Header[k] = vs
	}
	if c.httpAuth != nil {
		if err := c.httpAuth(httpReq.Header); err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, &httpError{
			StatusCode: httpResp.StatusCode,
			Body:       respBody,
		}
	}
	return respBody, nil
}

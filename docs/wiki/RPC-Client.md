# RPC Client

The typed JSON-RPC client lives in the **`rpc`** package. It is a
typed high-level client whose methods map one-to-one to Solana RPC
methods, built on top of the minimal transport in **`jsonrpc`**.

## Construction

Two construction styles, both forward to the same code path:

```go
import (
    "github.com/MevYu/solana-go/jsonrpc"
    "github.com/MevYu/solana-go/rpc"
)

// Struct-based options:
c := rpc.NewClient("https://api.mainnet-beta.solana.com", jsonrpc.Config{
    MaxIdleConnsPerHost: 20,
    Headers:             http.Header{"X-API-Key": {"secret"}},
})

// Functional options:
c := rpc.NewClientWith("https://api.mainnet-beta.solana.com",
    jsonrpc.WithMaxIdleConnsPerHost(20),
    jsonrpc.WithHeader("X-API-Key", "secret"),
)
```

To share a transport (HTTP client, codec, retry policy) across
multiple typed clients or alongside direct `*jsonrpc.Client.CallContext`
users:

```go
j := jsonrpc.NewClientWith(
    "https://api.mainnet-beta.solana.com",
    jsonrpc.WithHTTPClient(myHTTPClient),
    jsonrpc.WithCodec(jsonrpc.StdCodec()),
)
c := rpc.NewClientFromTransport(j)
```

The underlying transport is reachable via `c.Client` because
`*rpc.Client` embeds `*jsonrpc.Client`. To call an RPC method
that the typed layer does not yet wrap, use the generic
`jsonrpc.CallContext[T]` free function (Go forbids type
parameters on methods, so this lives as a free function rather
than a method on `*Client`):

```go
out, err := jsonrpc.CallContext[MyResult](ctx, c.Client, "myMethod", arg1, arg2)
```

For Solana's `{context:{slot}, value:T}` envelope, instantiate
`T = jsonrpc.ContextValue[X]` and read `out.Context.Slot` /
`out.Value`.

## Two layers

| Layer | Package | Purpose |
|---|---|---|
| Transport | `jsonrpc` | JSON-RPC 2.0 over HTTP, pluggable codec, retry policy, typed `*ErrRPC`, generic `CallContext[T]` + `ContextValue[T]` envelope, `Is*` classifiers |
| Typed wrappers | `rpc` | One Go method per RPC method with typed params and results — `cfg ...rpc.XxxCfg` instead of functional options |

The typed layer never imports `encoding/json` on the decode hot path
— the default codec is `goccy/go-json`, swappable via
`jsonrpc.WithCodec(jsonrpc.StdCodec())` if you need to drop the
dependency.

## Configuration

Both `jsonrpc.Config` (struct) and `WithXxx` (functional) options
support the same fields:

- **`HTTPClient *http.Client`** / **`WithHTTPClient(*http.Client)`** —
  replace the default `http.Client`. When set, `MaxIdleConns` and
  `MaxIdleConnsPerHost` are ignored. The default has **no global
  timeout**; bound requests via `context.WithTimeout` on every call.
- **`MaxIdleConns int`** / **`WithMaxIdleConns(n)`** — total idle
  connection cap across all hosts. Default: 100.
- **`MaxIdleConnsPerHost int`** / **`WithMaxIdleConnsPerHost(n)`** —
  per-host idle cap. Default: 10. Indexers and high-frequency bots
  typically want 50–200.
- **`Codec Codec`** / **`WithCodec(Codec)`** — swap the JSON codec.
  Default `GoJSONCodec()`; `StdCodec()` is available as an opt-out.
- **`RetryPolicy RetryPolicy`** / **`WithRetryPolicy(p)`** — swap the
  retry policy. Default exponential backoff with jitter, 5 attempts,
  200 ms base, 10 s cap; retries only transient errors (HTTP 5xx /
  429 / network errors).
- **`Headers http.Header`** / **`WithHeader(k,v)` /
  `WithHeaders(http.Header)`** — static request headers, applied to
  every call.
- **`HTTPAuth HTTPAuth`** / **`WithHTTPAuth(func(http.Header) error)`**
  — invoked before each request to inject dynamic auth headers
  (rotating API keys, signed challenges, etc.).

`(*jsonrpc.Client).SetHeader(key, value)` lets you update static
headers after construction (e.g. for rotating an API key). It is
**not** safe against concurrent in-flight requests; set it before
issuing requests or guard at the application level.

### `NewDefaultTransport`

Use `jsonrpc.NewDefaultTransport(maxIdleConns, maxIdleConnsPerHost)`
when you need to layer a custom RoundTripper (metrics, mTLS) on top
of the optimised transport rather than starting from
`http.DefaultTransport`:

```go
tr := jsonrpc.NewDefaultTransport(jsonrpc.DefaultMaxIdleConns, 50)
tr.TLSClientConfig = &tls.Config{...}
c := rpc.NewClientWith(url, jsonrpc.WithHTTPClient(&http.Client{Transport: tr}))
```

It returns an `*http.Transport` with `ProxyFromEnvironment`,
`ForceAttemptHTTP2: true`, `IdleConnTimeout: 90s`, and the dial /
TLS handshake timeouts pre-set to sensible Solana defaults.

## Typed method shape

Every typed method follows the same shape:

```go
func (c *Client) GetXxx(ctx context.Context, arg Foo, cfg ...rpc.GetXxxCfg) (*GetXxxResult, error)
```

- **`ctx`** — forwards to the underlying HTTP request and the
  retry loop; cancellation is respected during backoff.
- **`arg`** — typed (a `PublicKey`, a slot number, a signature,
  etc.).
- **`cfg`** — zero-or-one typed config struct (see
  [Call Options](Call-Options)). Every method documents the exact
  Cfg type it accepts, so options that don't apply are caught by the
  compiler instead of being silently ignored.
- **return** — a pointer to a typed result struct, or a sentinel
  like `uint64` / `solana.Signature` when that is the natural shape.
  Public result types carry json: tags but the wire-shape and the
  Go shape stay identical (single decode, no internal middleman).

## The context-value envelope

Solana RPC wraps many results in `{context:{slot}, value:T}`. The
`jsonrpc` package exposes the envelope type plus a single-pass
generic call so typed wrappers decode it without per-method
boilerplate:

```go
// Single-pass generic call. For envelope-wrapped responses,
// instantiate T = jsonrpc.ContextValue[X]:
env, err := jsonrpc.CallContext[jsonrpc.ContextValue[uint64]](
    ctx, c.Client, "getBalance", pubkey)
slot := env.Context.Slot
value := env.Value

// Envelope type, for callsites that own decoding (e.g. WebSocket
// notification dispatchers):
var n jsonrpc.ContextValue[*solana.AccountInfo]
_ = codec.Unmarshal(raw, &n)
```

`CallContext[T]` decodes the entire response — JSON-RPC envelope,
context slot, and typed value — in a **single** `codec.Unmarshal`
call. Methods whose response is not context-wrapped
(`getInflationRate`, `getEpochInfo`, …) instantiate `T` directly
on the result type.

## Errors

RPC errors from the server arrive as `*jsonrpc.ErrRPC`, wrapped with
the method name:

```go
var rpcErr *jsonrpc.ErrRPC
if errors.As(err, &rpcErr) {
    fmt.Println(rpcErr.Code, rpcErr.Msg)
    fmt.Println(rpcErr.Data) // method-specific, as jsonrpc.RawJSON
}
```

For pattern-matching on well-known failure modes
(`BlockhashNotFound`, node behind, rate limit, …), use the `Is*`
classifiers in the `jsonrpc` package. See
[Error Handling](Error-Handling).

## Method index

Covered methods, grouped by page:

- **Accounts** — [Account Methods](Account-Methods)
- **Blocks and tokens** — [Block & Token](Block-and-Token-Methods)
- **Chain info** — [Chain Info](Chain-Info-Methods)
- **Simple queries** — [Query Methods](Query-Methods)
- **Fees / airdrop** — [Fee Methods](Fee-Methods)
- **Sending transactions** — [Send Transaction](Send-Transaction)
- **Signature status / history** — [Signature Methods](Signature-Methods)
- **Simulation** — [Simulate Transaction](Simulate-Transaction)
- **Transaction retrieval** — [Transaction Query](Transaction-Query)
- **WebSocket subscriptions** — [WebSocket Client](WebSocket-Client)

For anything not yet covered typed, call through
`jsonrpc.CallContext[T](ctx, c.Client, method, params...)`
directly. For `{context, value}` responses, instantiate
`T = jsonrpc.ContextValue[X]`.

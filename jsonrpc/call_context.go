package jsonrpc

// ContextValue is the generic shape every Solana RPC method that
// returns a context-wrapped value takes on the wire:
//
//	{"context": {"slot": <slot>}, "value": <T>}
//
// Pass *ContextValue[T] as the result argument to Client.CallContext
// to decode this envelope; the slot is then available via
// resp.Context.Slot and the typed payload via resp.Value.
//
// The type is also reused by ws.Client notification dispatchers,
// which decode the same envelope shape from raw JSON bytes.
type ContextValue[T any] struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value T `json:"value"`
}

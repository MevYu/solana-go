package ws

// sendOrDropOldest is a generic non-blocking send that drops the
// oldest buffered element when the channel is full. It keeps the
// stream flowing when a slow consumer falls behind. T may be a pointer
// type (chan *Foo) or a value type (chan uint64).
func sendOrDropOldest[T any](ch chan T, v T) {
	select {
	case ch <- v:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- v:
		default:
		}
	}
}

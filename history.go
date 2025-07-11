package llmadapter

// History maintains the context of previous messages sent to or
// received from the LLM provider, to be able to send it with every
// request.
//
// It is generic in T, T being the content representation for any
// supported LLM provider.
type History[T any] struct {
	history []T
}

// Save records a message to the history.
func (h *History[T]) Save(message T) {
	h.history = append(h.history, message)
}

// Load loads the history to be used in a new request.
func (h *History[T]) Load() []T {
	return h.history
}

// Clear removes all history (including system instructions).
func (h *History[T]) Clear() {
	h.history = []T{}
}

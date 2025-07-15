package llmadapter

// History manages the conversation context by storing a sequence of messages.
// It is generic in type `T`, where `T` represents the specific message format
// required by a particular LLM provider (e.g., OpenAI's Message or AIStudio's ChatMessage).
// This allows the adapter to maintain conversation state across multiple requests.
type History[T any] struct {
	history map[*ThreadId][]T
}

// Save appends a new message to the conversation history.
// The `message` parameter should be of the generic type `T`, matching the
// content representation expected by the LLM provider.
func (h *History[T]) Save(threadId *ThreadId, message T) {
	if h.history == nil {
		h.history = make(map[*ThreadId][]T)
	}
	if _, ok := h.history[threadId]; !ok {
		h.history[threadId] = make([]T, 0)
	}

	h.history[threadId] = append(h.history[threadId], message)
}

// Load retrieves the entire conversation history as a slice of messages.
// This history can then be included in subsequent requests to the LLM
// to maintain conversational context.
func (h *History[T]) Load(threadId *ThreadId) []T {
	if h.history == nil {
		return []T{}
	}
	return h.history[threadId]
}

// Clear empties the entire conversation history, effectively starting a
// new conversation. This also clears any system instructions that were
// part of the history.
func (h *History[T]) Clear(threadId *ThreadId) {
	if h.history != nil {
		delete(h.history, threadId)
	}
}

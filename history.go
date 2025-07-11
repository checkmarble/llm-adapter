package llmadapter

type History[T any] struct {
	history []T
	len     int
}

func (h *History[T]) Save(message T) {
	h.history = append(h.history, message)
}

func (h *History[T]) Load() []T {
	return h.history
}

func (h *History[T]) Clear() {
	h.history = []T{}
}

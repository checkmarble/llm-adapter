package llmadapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveLoadHistory(t *testing.T) {
	type message struct {
		Number int
	}

	h := History[message]{
		history: make(map[*ThreadId][]message),
	}

	assert.Len(t, h.history, 0)

	threadId := &ThreadId{}

	h.Save(threadId, message{1})
	h.Save(threadId, message{2})
	h.Save(threadId, message{3})

	assert.Len(t, h.Load(threadId), 3)
	assert.ElementsMatch(t, h.Load(threadId), []message{{1}, {2}, {3}})
}
func TestClearHistory(t *testing.T) {
	type message struct {
		Number int
	}

	h := History[message]{
		history: make(map[*ThreadId][]message),
	}

	threadId := &ThreadId{}

	assert.Len(t, h.Load(threadId), 0)

	h.Save(threadId, message{1})
	h.Save(threadId, message{2})
	h.Save(threadId, message{3})

	assert.Len(t, h.Load(threadId), 3)
	assert.ElementsMatch(t, h.Load(threadId), []message{{1}, {2}, {3}})

	h.Clear(threadId)

	assert.Len(t, h.Load(threadId), 0)
}

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
		history: make([]message, 0),
	}

	assert.Len(t, h.Load(), 0)

	h.Save(message{1})
	h.Save(message{2})
	h.Save(message{3})

	assert.Len(t, h.Load(), 3)
	assert.ElementsMatch(t, h.Load(), []message{{1}, {2}, {3}})
}
func TestClearHistory(t *testing.T) {
	type message struct {
		Number int
	}

	h := History[message]{
		history: make([]message, 0),
	}

	assert.Len(t, h.Load(), 0)

	h.Save(message{1})
	h.Save(message{2})
	h.Save(message{3})

	assert.Len(t, h.Load(), 3)
	assert.ElementsMatch(t, h.Load(), []message{{1}, {2}, {3}})

	h.Clear()

	assert.Len(t, h.Load(), 0)
}

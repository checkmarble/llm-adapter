package llmadapter

import (
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSyncAll(t *testing.T) {
	e := errors.New("error")

	p1 := NewMockProvider()
	p1.On("Init", mock.Anything).Return(nil)

	llm, _ := New(WithDefaultProvider(p1))

	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"first"}, nil).After(100 * time.Millisecond).Once()
	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(nil, e).After(300 * time.Millisecond).Once()
	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"third"}, nil).After(500 * time.Millisecond).Once()

	responses := All(t.Context(), llm,
		NewUntypedRequest(),
		NewUntypedRequest(),
		NewUntypedRequest())

	assert.Len(t, responses, 3)

	values := lo.Map(responses, func(resp AsyncResponse[string], _ int) any {
		if resp.Error != nil {
			return resp.Error
		}

		output, err := resp.Response.Get(0)

		assert.Nil(t, err)

		return output
	})

	assert.ElementsMatch(t, values, []any{"first", e, "third"})
}

func TestSyncRace(t *testing.T) {
	p1 := NewMockProvider()
	p1.On("Init", mock.Anything).Return(nil)

	llm, _ := New(WithDefaultProvider(p1))

	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(nil, errors.New("error")).Once()
	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"first"}, nil).After(200 * time.Millisecond).Once()
	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"second"}, nil).After(400 * time.Millisecond).Once()
	p1.On("ChatCompletion", mock.Anything, llm, mock.Anything).Return(MockMessage{"third"}, nil).After(500 * time.Millisecond).Once()

	resp, err := Race(t.Context(), llm,
		NewUntypedRequest(),
		NewUntypedRequest(),
		NewUntypedRequest())

	assert.Nil(t, err)
	assert.NotNil(t, resp)

	output, err := resp.Get(0)

	assert.Nil(t, err)
	assert.Equal(t, "first", output)
}

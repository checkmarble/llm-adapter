package llmadapter

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
)

type AsyncResponse[T any] struct {
	Response *Response[T]
	Error    error
}

func All[T any](ctx context.Context, llm *LlmAdapter, reqs ...Request[T]) []AsyncResponse[T] {
	var wg sync.WaitGroup

	responses := make([]AsyncResponse[T], len(reqs))

	for idx, req := range reqs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			resp, err := req.Do(ctx, llm)
			if err != nil {
				responses[idx] = AsyncResponse[T]{Error: err}
				return
			}

			responses[idx] = AsyncResponse[T]{Response: resp}
		}()
	}

	wg.Wait()

	return responses
}

func Race[T any](ctx context.Context, llm *LlmAdapter, reqs ...Request[T]) AsyncResponse[T] {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := make(chan AsyncResponse[T], len(reqs))

	for _, req := range reqs {
		go func() {
			resp, err := req.Do(ctx, llm)
			if err != nil {
				c <- AsyncResponse[T]{Error: err}
				return
			}

			c <- AsyncResponse[T]{Response: resp}
		}()
	}

	errored := 0

	for {
		select {
		case <-ctx.Done():
			return AsyncResponse[T]{Error: ctx.Err()}

		case value := <-c:
			switch value.Error {
			case nil:
				return value

			default:
				errored += 1

				if errored == len(reqs) {
					return AsyncResponse[T]{Error: errors.New("all requests failed")}
				}
			}
		}
	}
}

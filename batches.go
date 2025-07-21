package llmadapter

import (
	"context"

	"github.com/checkmarble/marble-llm-adapter/internal"
	"github.com/cockroachdb/errors"
)

type (
	BatchStatus int
)

const (
	BatchPending BatchStatus = iota
	BatchRunning
	BatchFinished
	BatchError
)

type BatchUnsupported struct{}

func (BatchUnsupported) SubmitBatch(ctx context.Context, llm internal.Adapter, reqs ...Requester) (*UntypedBatchPromise, error) {
	return nil, errors.New("provider does not support batch mode")
}

func (BatchUnsupported) Check(context.Context, *UntypedBatchPromise) (BatchStatus, error) {
	return BatchError, errors.New("provider does not support batch mode")
}

func (BatchUnsupported) Wait(ctx context.Context, pr *UntypedBatchPromise) <-chan BatchWaitResponse {
	return nil
}

type Batch[T any] struct {
	Requests []Request[T]
}

func (b Batch[T]) Batch(ctx context.Context, llm *LlmAdapter, providerName string) (*BatchPromise[T], error) {
	requesters := make([]Requester, len(b.Requests))

	for idx, r := range b.Requests {
		requesters[idx] = Requester(r)
	}

	promise, err := llm.SubmitBatch(ctx, providerName, requesters...)

	if err != nil {
		return nil, err
	}

	return &BatchPromise[T]{promise}, nil
}

type UntypedBatchPromise struct {
	Provider     Llm
	ProviderName string
	Id           string
}

type BatchPromise[T any] struct {
	*UntypedBatchPromise
}

func (p BatchPromise[T]) Check(ctx context.Context) (BatchStatus, error) {
	return p.Provider.Check(ctx, p.UntypedBatchPromise)
}

func (p BatchPromise[T]) Wait(ctx context.Context) (map[string]Response[T], error) {
	inners := <-p.Provider.Wait(ctx, p.UntypedBatchPromise)

	if inners.Error != nil {
		return nil, inners.Error
	}

	responses := make(map[string]Response[T], len(inners.Responses))

	for id, resp := range inners.Responses {
		responses[id] = Response[T]{
			InnerResponse: resp,
		}
	}

	return responses, nil
}

type BatchWaitResponse struct {
	Status   BatchStatus
	Filename string
	Error    error

	Responses map[string]InnerResponse
}

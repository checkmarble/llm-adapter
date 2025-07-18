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

func (BatchUnsupported) SubmitBatch(ctx context.Context, llm internal.Adapter, reqs ...Requester) (*BatchPromise, error) {
	return nil, errors.New("provider does not support batch mode")
}

func (BatchUnsupported) Check(context.Context, *BatchPromise) (BatchStatus, error) {
	return BatchError, errors.New("provider does not support batch mode")
}

func (BatchUnsupported) Wait(ctx context.Context, pr *BatchPromise) <-chan BatchWaitResponse {
	return nil
}

type BatchRequest struct {
	Provider Llm
	Filename string
}

type BatchPromise struct {
	Provider     Llm
	ProviderName string
	Id           string
}

func (p *BatchPromise) Check(ctx context.Context) (BatchStatus, error) {
	return p.Provider.Check(ctx, p)
}

func (p *BatchPromise) Wait(ctx context.Context) <-chan BatchWaitResponse {
	return p.Provider.Wait(ctx, p)
}

type BatchWaitResponse struct {
	Status BatchStatus
	Error  error
}

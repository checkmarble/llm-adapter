package aistudio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/internal"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/simonfrey/jsonl"
	"google.golang.org/genai"
)

type BatchPayload struct {
	Key     string               `json:"key"`
	Request genai.InlinedRequest `json:"request"`
}

func (p *AiStudio) SubmitBatch(ctx context.Context, llm internal.Adapter, reqs ...llmadapter.Requester) (*llmadapter.BatchPromise, error) {
	if p.name == "" {
		return nil, errors.New("batches can only be created on named providers")
	}

	payload, err := p.createBatchInput(llm, reqs...)
	if err != nil {
		return nil, err
	}

	req, err := p.uploadFile(ctx, payload)
	if err != nil {
		return nil, err
	}

	src := genai.BatchJobSource{}

	switch p.backend {
	case genai.BackendGeminiAPI:
		src.FileName = req.Filename
	case genai.BackendVertexAI:
		src.Format = "jsonl"
		src.GCSURI = []string{req.Filename}
	}

	job, err := p.client.Batches.Create(
		ctx, lo.FromPtr(lo.CoalesceOrEmpty(p.model, lo.ToPtr(llm.DefaultModel()))),
		&src,
		&genai.CreateBatchJobConfig{
			DisplayName: "Created on " + time.Now().Format(time.RFC3339),
			Dest: &genai.BatchJobDestination{
				Format: "jsonl",
				GCSURI: fmt.Sprintf("gs://%s/llm/outputs", p.bucket),
			},
		})

	if err != nil {
		return nil, err
	}

	return &llmadapter.BatchPromise{
		Provider:     p,
		ProviderName: p.name,
		Id:           job.Name,
	}, nil
}

func (p *AiStudio) createBatchInput(llm internal.Adapter, requesters ...llmadapter.Requester) (io.Reader, error) {
	if p.name == "" {
		return nil, errors.New("batches can only be created on named providers")
	}

	var buf bytes.Buffer

	w := jsonl.NewWriter(&buf)

	for _, requester := range requesters {
		id := requester.ToRequest().Id

		if requester.Error() != nil {
			return nil, requester.Error()
		}

		if id == "" {
			return nil, errors.New("all requests in a batch must have an ID")
		}

		model, ok := lo.Coalesce(requester.ToRequest().Model, p.model, lo.ToPtr(llm.DefaultModel()))
		if !ok {
			return nil, errors.New("no model was configured")
		}

		opts := internal.CastProviderOptions[RequestOptions](requester.ProviderRequestOptions(p))

		contents, cfg, err := p.adaptRequest(llm, requester, opts)
		if err != nil {
			return nil, err
		}

		payload := BatchPayload{
			Key: id,
			Request: genai.InlinedRequest{
				Model:    *model,
				Config:   cfg,
				Contents: contents,
			},
		}

		if err := w.Write(payload); err != nil {
			return nil, err
		}
	}

	return &buf, nil
}

func (p *AiStudio) Check(ctx context.Context, pr *llmadapter.BatchPromise) (llmadapter.BatchStatus, error) {
	job, err := p.client.Batches.Get(ctx, pr.Id, nil)
	if err != nil {
		return llmadapter.BatchError, err
	}

	return adaptJobState(job.State), nil
}

func (p *AiStudio) Wait(ctx context.Context, pr *llmadapter.BatchPromise) <-chan llmadapter.BatchWaitResponse {
	ch := make(chan llmadapter.BatchWaitResponse)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return

			default:
				job, err := p.client.Batches.Get(ctx, pr.Id, nil)
				if err != nil {
					ch <- llmadapter.BatchWaitResponse{Error: err}
					close(ch)
					return
				}

				if !job.EndTime.IsZero() {
					ch <- llmadapter.BatchWaitResponse{Status: adaptJobState(job.State)}
					close(ch)
					return
				}
			}

			time.Sleep(30 * time.Second)
		}
	}()

	return ch
}
func (p *AiStudio) uploadFile(ctx context.Context, r io.Reader) (*llmadapter.BatchRequest, error) {
	if p.name == "" {
		return nil, errors.New("batches can only be created on named providers")
	}

	switch p.backend {
	case genai.BackendGeminiAPI:
		file, err := p.client.Files.Upload(ctx, r, &genai.UploadFileConfig{
			MIMEType: "jsonl",
		})

		if err != nil {
			return nil, err
		}

		return &llmadapter.BatchRequest{
			Provider: p,
			Filename: file.Name,
		}, nil

	case genai.BackendVertexAI:
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, err
		}

		filename := fmt.Sprintf("llm/inputs/%s", "input.jsonl")
		object := client.Bucket(p.bucket).Object(filename)
		wr := object.NewWriter(ctx)

		if _, err := io.Copy(wr, r); err != nil {
			return nil, err
		}
		if err := wr.Close(); err != nil {
			return nil, err
		}

		return &llmadapter.BatchRequest{
			Provider: p,
			Filename: fmt.Sprintf("gs://%s/%s", p.bucket, filename),
		}, nil

	default:
		return nil, errors.New("invalid backend")
	}
}

func adaptJobState(state genai.JobState) llmadapter.BatchStatus {
	switch state {
	case genai.JobStatePending:
		return llmadapter.BatchPending
	case genai.JobStateCancelled, genai.JobStateSucceeded, genai.JobStatePartiallySucceeded:
		return llmadapter.BatchFinished
	case genai.JobStateRunning:
		return llmadapter.BatchRunning
	default:
		return llmadapter.BatchPending
	}
}

package perplexity

import (
	"reflect"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/internal"
	base "github.com/checkmarble/marble-llm-adapter/llms/openai"
	"github.com/fatih/structs"
	"github.com/openai/openai-go"
)

type Perplexity struct {
	*base.OpenAi
}

func (*Perplexity) RequestOptionsType() reflect.Type {
	return reflect.TypeFor[RequestOptions]()
}

func New(openAiOpts ...base.Opt) (*Perplexity, error) {
	oai, err := base.New(
		base.WithBaseUrl("https://api.perplexity.ai"),
	)

	if err != nil {
		return nil, err
	}

	for _, opt := range openAiOpts {
		opt(oai)
	}

	llm := Perplexity{
		OpenAi: oai,
	}

	llm.RequestHookFunc = llm.transformRequest

	return &llm, nil
}

func (p *Perplexity) transformRequest(requester llmadapter.Requester, cfg *openai.ChatCompletionNewParams) error {
	opts := internal.CastProviderOptions[RequestOptions](requester.ProviderRequestOptions(p))

	cfg.SetExtraFields(structs.Map(opts))

	return nil
}

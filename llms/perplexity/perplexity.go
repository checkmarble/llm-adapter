package perplexity

import (
	"encoding/json"
	"reflect"
	"time"

	llmadapter "github.com/checkmarble/llm-adapter"
	"github.com/checkmarble/llm-adapter/internal"
	base "github.com/checkmarble/llm-adapter/llms/openai"
	"github.com/fatih/structs"
	"github.com/openai/openai-go"
	"github.com/samber/lo"
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
	llm.ResponseHookFunc = llm.transformResponse

	return &llm, nil
}

func (p *Perplexity) transformRequest(requester llmadapter.Requester, cfg *openai.ChatCompletionNewParams) error {
	opts := internal.CastProviderOptions[RequestOptions](requester.ProviderRequestOptions(p))

	cfg.SetExtraFields(structs.Map(opts))

	return nil
}

func (p *Perplexity) transformResponse(response *openai.ChatCompletion, resp *llmadapter.InnerResponse) error {
	searchResultsField, ok := response.JSON.ExtraFields["search_results"]
	if !ok {
		return nil
	}

	searchResultsContent := searchResultsField.Raw()

	searchResults := []SearchResult{}

	if err := json.Unmarshal([]byte(searchResultsContent), &searchResults); err != nil {
		return err
	}

	// Loop thought all candidates
	// Perplexity returns only one candidate, the search results are linked to this candidate
	// Use loop to avoid dealing with checking if there is a candidate or not
	for i := range resp.Candidates {
		grouding := llmadapter.ResponseGrounding{
			Sources: lo.Map(searchResults, func(result SearchResult, _ int) llmadapter.ResponseGroudingSource {
				date, _ := time.Parse(time.DateOnly, result.Date)

				return llmadapter.ResponseGroudingSource{
					Title: result.Title,
					Url:   result.URL,
					Date:  date,
				}
			}),
		}
		resp.Candidates[i].Grounding = &grouding
	}

	return nil
}

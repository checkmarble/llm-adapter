package perplexity

import (
	"io"
	"net/http"
	"testing"
	"time"

	llmadapter "github.com/checkmarble/marble-llm-adapter"
	"github.com/checkmarble/marble-llm-adapter/llms/openai"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestPerplexityExtras(t *testing.T) {
	defer gock.Off()

	provider, _ := openai.New(openai.WithApiKey("apikey"))
	llm, _ := llmadapter.New(llmadapter.WithDefaultProvider(provider))

	req := llmadapter.NewUntypedRequest().
		WithProviderOptions(openai.RequestOptions{
			Extras: Params{
				SearchMode: SearchModeAcademic,
				BeforeDate: NewDate(time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)),
				WebSearch: &WebSearch{
					UserLocation: &UserLocation{Country: "FR"},
				},
			},
		})

	gock.New("https://api.openai.com").
		Post("/v1/chat/completions").
		MatchHeader("authorization", "Bearer apikey").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			body, _ := io.ReadAll(req.Body)

			assert.Equal(t, "academic", gjson.GetBytes(body, "search_mode").String())
			assert.Equal(t, "7/1/2025", gjson.GetBytes(body, "search_before_date_filter").String())
			assert.Equal(t, "FR", gjson.GetBytes(body, "web_search_options.user_location.country").String())

			return true, nil
		}).
		Reply(http.StatusOK).
		SetHeader("content-type", "application/json")

	_, _ = req.Do(t.Context(), llm)

	assert.False(t, gock.HasUnmatchedRequest())
}

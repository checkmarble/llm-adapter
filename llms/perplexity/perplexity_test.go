package perplexity

import (
	"io"
	"net/http"
	"testing"
	"time"

	llmberjack "github.com/checkmarble/llmberjack"
	"github.com/checkmarble/llmberjack/llms/openai"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestPerplexityExtras(t *testing.T) {
	defer gock.Off()

	provider, _ := New(openai.WithApiKey("apikey"))
	llm, _ := llmberjack.New(llmberjack.WithDefaultProvider(provider))

	req := llmberjack.NewUntypedRequest().
		WithProviderOptions(RequestOptions{
			SearchMode: SearchModeAcademic,
			BeforeDate: NewDate(time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)),
			WebSearch: WebSearch{
				UserLocation: UserLocation{Country: "FR"},
			},
			SearchDomainFilter: []string{"google.com", "wikipedia.org"},
		}).
		WithText(llmberjack.RoleUser, "C'est une bonne situation ça Scribe ?")

	gock.New("https://api.perplexity.ai").
		Post("/chat/completions").
		MatchHeader("authorization", "Bearer apikey").
		AddMatcher(func(req *http.Request, _ *gock.Request) (bool, error) {
			body, _ := io.ReadAll(req.Body)

			assert.Len(t, gjson.GetBytes(body, "@this").Map(), 5) // Always includes `messages`.
			assert.Equal(t, "academic", gjson.GetBytes(body, "search_mode").String())
			assert.Equal(t, "7/1/2025", gjson.GetBytes(body, "search_before_date_filter").String())
			assert.Equal(t, "FR", gjson.GetBytes(body, "web_search_options.user_location.country").String())
			assert.Len(t, gjson.GetBytes(body, "search_domain_filter").Array(), 2)
			assert.Equal(t, "google.com", gjson.GetBytes(body, "search_domain_filter").Array()[0].String())
			assert.Equal(t, "wikipedia.org", gjson.GetBytes(body, "search_domain_filter").Array()[1].String())
			assert.Equal(t, "C'est une bonne situation ça Scribe ?", gjson.GetBytes(body, "messages.0.content.0.text").String())
			return true, nil
		}).
		Reply(http.StatusOK).
		SetHeader("content-type", "application/json").
		BodyString(`
		{
			"id": "Numerobis",
			"model": "sonar",
			"created": 1756219624,
			"usage": {
				"prompt_tokens": 79,
				"completion_tokens": 367,
				"total_tokens": 446,
				"search_context_size": "low",
				"cost": {
					"input_tokens_cost": 0,
					"output_tokens_cost": 0,
					"request_cost": 0.005,
					"total_cost": 0.005
				}
			},
			"object": "chat.completion",
			"choices": [
				{
					"index": 0,
					"finish_reason": "stop",
					"message": {
						"content": "Vous savez, moi je ne crois pas qu’il y ait de bonne ou de mauvaise situation.",
						"role": "system"
					}
				}
			],
			"search_results": [
				{
					"title": "Je n'en peux plus de ces papyrus là",
					"url": "https://www.kaakook.fr/citation-1331",
					"date": "2002-01-30"
				},
				{
					"title": "Le mec… Il s’appelle On ! Donc c’est le phare-à-On ! Le pharaon !",
					"url": "https://www.kaakook.fr/citation-4422",
					"date": "2002-01-30"
				}
			]
		}
		`)

	resp, err := req.Do(t.Context(), llm)

	assert.NoError(t, err)
	assert.Equal(t, "Vous savez, moi je ne crois pas qu’il y ait de bonne ou de mauvaise situation.", resp.Candidates[0].Text)
	assert.Equal(t, "Je n'en peux plus de ces papyrus là", resp.Candidates[0].Grounding.Sources[0].Title)
	assert.Equal(t, "https://www.kaakook.fr/citation-1331", resp.Candidates[0].Grounding.Sources[0].Url)
	date, _ := time.Parse(time.DateOnly, "2002-01-30")
	assert.Equal(t, date, resp.Candidates[0].Grounding.Sources[0].Date)
	assert.Equal(t, "Le mec… Il s’appelle On ! Donc c’est le phare-à-On ! Le pharaon !", resp.Candidates[0].Grounding.Sources[1].Title)
	assert.Equal(t, "https://www.kaakook.fr/citation-4422", resp.Candidates[0].Grounding.Sources[1].Url)
	assert.Equal(t, date, resp.Candidates[0].Grounding.Sources[1].Date)

	assert.False(t, gock.HasUnmatchedRequest())
}

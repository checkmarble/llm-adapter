package openai

type opt func(*OpenAi)

// WithBaseUrl sets the URL at which the OpenAI-compatible API is available.
//
// If not specified, will use OpenAI's API.
func WithBaseUrl(url string) opt {
	return func(p *OpenAi) {
		p.baseUrl = url
	}
}

func WithApiKey(apiKey string) opt {
	return func(p *OpenAi) {
		p.apiKey = apiKey
	}
}

func WithDefaultModel(model string) opt {
	return func(p *OpenAi) {
		p.model = &model
	}
}

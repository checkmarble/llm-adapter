package openai

type llmOption func(*OpenAi)

func WithBaseUrl(url string) llmOption {
	return func(p *OpenAi) {
		p.baseUrl = url
	}
}

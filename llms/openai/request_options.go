package openai

type Extras interface {
	Extras()
}

type RequestOptions struct {
	Extras Extras
}

func (RequestOptions) ProviderRequestOptions() {}

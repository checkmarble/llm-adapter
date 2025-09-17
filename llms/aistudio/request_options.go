package aistudio

type ThinkingConfig struct {
	IncludeThoughts bool
	// To disable thinking, set the budget to 0
	// cf: https://cloud.google.com/vertex-ai/generative-ai/docs/thinking#budget
	Budget *int32
}

type RequestOptions struct {
	GoogleSearch *bool
	TopK         *float64
	Thinking     *ThinkingConfig
}

func (RequestOptions) ProviderRequestOptions() {}

package aistudio

type RequestOptions struct {
	GoogleSearch *bool
	TopK         *float64
}

func (RequestOptions) RequestOptionsForProvider() {}

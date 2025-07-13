package internal

import "net/http"

type Adapter interface {
	DefaultModel() string
	ApiKey() string
	SaveContext() bool
	HttpClient() *http.Client
}

type ProviderRequestOptions interface {
	RequestOptionsForProvider()
}

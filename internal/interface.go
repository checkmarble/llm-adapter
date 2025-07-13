package internal

type Adapter interface {
	DefaultModel() string
	ApiKey() string
	SaveContext() bool
}

type ProviderRequestOptions interface {
	RequestOptionsForProvider()
}

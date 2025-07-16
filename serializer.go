package llmadapter

type LlmSerializer[T any] interface {
	Serialize(input any) (T, error)
}

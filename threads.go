package llmadapter

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type ThreadId struct {
	_        noCopy
	provider Llm
}

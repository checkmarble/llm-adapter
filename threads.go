package llmadapter

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

type ThreadId struct {
	_        noCopy
	provider Llm
}

func (t *ThreadId) Clear() {
	t.provider.ResetThread(t)
}

func (t *ThreadId) Copy() *ThreadId {
	return t.provider.CopyThread(t)
}

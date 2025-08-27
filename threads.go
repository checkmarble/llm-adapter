package llmadapter

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// ThreadId uniquely represents a conversation with an LLM.
//
// It is used to mark and identify a specific conversation and accumulate its
// history. ThreadsId inherent identifiers (their memory address is the
// identifier), so their value cannot be copied, only pointers should be passed
// around.
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

func (t *ThreadId) Close() {
	t.provider.CloseThread(t)
}

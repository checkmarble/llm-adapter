package llmadapter

type Response struct {
	Model      string
	Candidates []ResponseCandidate
}

type ResponseCandidate struct {
	Text []string
}

package llmadapter

import "io"

type (
	MessageRole int
	MessageType string
)

const (
	RoleSystem MessageRole = iota
	RoleUser
	RoleAi
)

const (
	TypeText = "text/plain"
)

type Request struct {
	Model    *string
	Messages []Message
}

func NewRequest() Request {
	return Request{
		Messages: make([]Message, 0),
	}
}

func (r Request) WithModel(model string) Request {
	r.Model = &model

	return r
}

func (r Request) WithSystemInstruction(parts ...io.Reader) Request {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  RoleSystem,
		Parts: parts,
	})

	return r
}

func (r Request) WithText(role MessageRole, parts ...io.Reader) Request {
	r.Messages = append(r.Messages, Message{
		Type:  TypeText,
		Role:  role,
		Parts: parts,
	})

	return r
}

type Message struct {
	Type  MessageType
	Role  MessageRole
	Parts []io.Reader
}

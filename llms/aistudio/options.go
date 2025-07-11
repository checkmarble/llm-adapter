package aistudio

import "google.golang.org/genai"

type opt func(*AiStudio)

// WithBackend represents which Google GenAI backend to use (VertexAI or Gemini).
//
// It only accepts values of `genai.GeminiAPI` or `genai.VertexAI`.
func WithBackend(backend genai.Backend) opt {
	return func(p *AiStudio) {
		p.backend = backend
	}
}

// WithProject defines the Google Cloud Platform project to use to connect to VertexAI.
//
// It is only taken into account when using the VertexAI backend.
func WithProject(project string) opt {
	return func(p *AiStudio) {
		p.project = project
	}
}

// WithLocation defines the Google Cloud Platform region to use to connect to VertexAI.
//
// It is only taken into account when using the VertexAI backend.
func WithLocation(location string) opt {
	return func(p *AiStudio) {
		p.location = location
	}
}

func WithDefaultModel(model string) opt {
	return func(p *AiStudio) {
		p.model = &model
	}
}

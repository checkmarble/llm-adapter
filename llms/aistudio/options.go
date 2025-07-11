package aistudio

import "google.golang.org/genai"

type opt func(*AiStudio)

func WithBackend(backend genai.Backend) opt {
	return func(p *AiStudio) {
		p.backend = backend
	}
}

func WithProject(project string) opt {
	return func(p *AiStudio) {
		p.project = project
	}
}

func WithLocation(location string) opt {
	return func(p *AiStudio) {
		p.location = location
	}
}

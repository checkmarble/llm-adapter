package aistudio

type opt func(*AiStudio)

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

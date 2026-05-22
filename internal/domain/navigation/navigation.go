package navigation

import "fmt"

func PublicationPath(topicSlug, publicationSlug string, pageNumber int) string {
	if pageNumber <= 1 {
		return fmt.Sprintf("%s/%s/", topicSlug, publicationSlug)
	}
	return fmt.Sprintf("%s/%s-%d/", topicSlug, publicationSlug, pageNumber)
}

type State struct {
	Current string
	Next    string
	Start   string
	IsLast  bool
}

func Build(topicSlug, publicationSlug string, pageNumber, totalPages int) State {
	current := PublicationPath(topicSlug, publicationSlug, pageNumber)
	state := State{
		Current: current,
		Start:   PublicationPath(topicSlug, publicationSlug, 1),
		IsLast:  pageNumber >= totalPages,
	}
	if pageNumber < totalPages {
		state.Next = PublicationPath(topicSlug, publicationSlug, pageNumber+1)
	}
	return state
}

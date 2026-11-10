package graph

import "github.com/99designs/gqlgen/_examples/federation/v1/content/graph/model"

// Resolver serves one video and one article.
type Resolver struct{}

func videoByID(id string) *model.Video {
	if id != "video-1" {
		return nil
	}
	publishedOn := "2024-01-02"
	return &model.Video{
		ID:              id,
		Title:           "Adding fractions",
		DurationSeconds: 300,
		PublishedOn:     &publishedOn,
	}
}

func articleByID(id string) *model.Article {
	if id != "article-1" {
		return nil
	}
	return &model.Article{ID: id, Title: "Decimals", WordCount: 1200}
}

package models

import (
	"time"
)

type Article struct {
	Id       string
	Category string
	Title    string
	Subtitle string
	Content  string
	Login    string
	Created  time.Time
	User_id  int
}

func NewArticle(category string, title string, subtitle string, content string, login string) *Article {
	return &Article{
		"nil",
		category,
		title,
		subtitle,
		content,
		login,
		time.Now(),
		0,
	}
}
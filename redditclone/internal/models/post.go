package models

import (
	"errors"
	"math/rand"
	"strconv"
	"time"
)

var (
	ErrNoVotes = errors.New("there is no votes on the post")
	ErrNoComm  = errors.New("can not find comment on the post")
)

type Vote struct {
	User string `json:"user"`
	Vote int    `json:"vote"`
}

type Post struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Type          string    `json:"type"`
	Category      string    `json:"category"`
	Score         int       `json:"score"`
	Text          string    `json:"text,omitempty"`
	Url           string    `json:"url,omitempty"`
	UpvotePercent int       `json:"upvotePercentage"`
	Views         int       `json:"views"`
	Comments      []Comment `json:"comments"`
	Votes         []Vote    `json:"votes"`
	CreatedAt     string    `json:"created"`
	Author        *User     `json:"author"`
}

func NewPost(title, category, pType, text, url string, author *User) *Post {
	return &Post{
		ID:        strconv.FormatInt(rand.Int63(), 10),
		Title:     title,
		Type:      pType,
		Category:  category,
		Text:      text,
		Url:       url,
		Author:    author,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func (p *Post) Vote(u *User, value int) {
	if len(p.Votes) <= 0 {
		p.Votes = append(p.Votes, Vote{User: u.ID, Vote: value})
		return
	}

	for i, vote := range p.Votes {
		if vote.User == u.ID {
			p.Votes[i].Vote = value
			return
		}
	}

	p.Votes = append(p.Votes, Vote{User: u.ID, Vote: value})
}

func (p *Post) Unvote(u *User) error {
	if len(p.Votes) <= 0 {
		return ErrNoVotes
	}

	index := -1
	for i, vote := range p.Votes {
		if vote.User == u.ID {
			index = i
			break
		}
	}

	if index == -1 {
		return ErrNoVotes
	}

	p.Votes = append(p.Votes[:index], p.Votes[index+1:]...)

	return nil
}

// TODO(illusion): in some future let's optimize this by moving into the cycles on Vote and Unvote methods
func (p *Post) RecalculateScore() {
	p.Score = 0

	if len(p.Votes) == 0 {
		p.UpvotePercent = 0
		return
	}

	upvotes := 0

	for _, vote := range p.Votes {
		p.Score += vote.Vote

		if vote.Vote > 0 {
			upvotes++
		}
	}

	p.UpvotePercent = upvotes * 100 / len(p.Votes)
}

func (p *Post) View() *Post {
	p.Views++
	return p
}

func (p *Post) AddComment(body string, author *User) {
	p.Comments = append(p.Comments, Comment{
		ID:        strconv.FormatInt(rand.Int63(), 10),
		Body:      body,
		Author:    author,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (p *Post) DeleteComment(id string) error {
	index := -1
	for i, comment := range p.Comments {
		if comment.ID == id {
			index = i
		}
	}

	if index == -1 {
		return ErrNoComm
	}

	p.Comments = append(p.Comments[:index], p.Comments[index+1:]...)
	return nil
}

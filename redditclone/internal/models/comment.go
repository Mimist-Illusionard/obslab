package models

type Comment struct {
	ID        string `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created"`
	Author    *User  `json:"author"`
}

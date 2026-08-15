package repository

import (
	"errors"
	"sync"

	"github.com/Mimist-Illusionard/obslab/internal/models"
)

var (
	ErrNoPost = errors.New("post doesn't exist")
)

type PostRepository interface {
	List(category string) []models.Post
	ListByUser(login string) ([]models.Post, error)
	Create(title, category, pType, text, url string, author *models.User) (*models.Post, error)
	Get(id string) (*models.Post, error)
	Save(post *models.Post) error
	DeleteComment(postId, commentId string) (*models.Post, error)
	Delete(postId string) error
}

type PostMemoryRepository struct {
	posts []models.Post
	mu    sync.Mutex
}

func NewPostMemoryRepository() *PostMemoryRepository {
	return &PostMemoryRepository{posts: make([]models.Post, 0)}
}

func (r *PostMemoryRepository) List(category string) []models.Post {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]models.Post, 0)
	if category == "" {
		return append([]models.Post(nil), r.posts...)
	}

	for _, post := range r.posts {
		if post.Category == category {
			result = append(result, post)
		}
	}

	return result
}

func (r *PostMemoryRepository) Create(title, category, pType, text, url string, author *models.User) (*models.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p := models.NewPost(title, category, pType, text, url, author)
	p.Vote(author, 1)
	p.RecalculateScore()

	r.posts = append(r.posts, *p)

	return p, nil
}

func (r *PostMemoryRepository) Get(id string) (*models.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID == id {
			post := r.posts[i]
			return &post, nil
		}
	}

	return nil, ErrNoPost
}

func (r *PostMemoryRepository) DeleteComment(postID, commentID string) (*models.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i, post := range r.posts {
		if post.ID != postID {
			continue
		}

		err := r.posts[i].DeleteComment(commentID)
		if err != nil {
			return nil, err
		}

		index = i
	}

	if index == -1 {
		return nil, ErrNoPost
	}

	return &r.posts[index], nil
}

func (r *PostMemoryRepository) Save(post *models.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.posts {
		if r.posts[i].ID == post.ID {
			r.posts[i] = *post
			return nil
		}
	}

	return ErrNoPost
}

func (r *PostMemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i := range r.posts {
		if r.posts[i].ID == id {
			index = i
		}
	}

	if index == -1 {
		return ErrNoPost
	}

	r.posts = append(r.posts[:index], r.posts[index+1:]...)
	return nil
}

func (r *PostMemoryRepository) ListByUser(login string) ([]models.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]models.Post, 0)
	for _, post := range r.posts {
		if post.Author.Name == login {
			result = append(result, post)
		}
	}

	return result, nil
}

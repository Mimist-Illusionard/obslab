package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Mimist-Illusionard/obslab/internal/dto"
	"github.com/Mimist-Illusionard/obslab/internal/jwt"
	"github.com/Mimist-Illusionard/obslab/internal/models"
	"github.com/Mimist-Illusionard/obslab/internal/repository"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type PostHandler struct {
	r      repository.PostRepository
	jwtGen *jwt.JwtService
	logger *zap.Logger
}

func NewPostHandler(r repository.PostRepository, logger *zap.Logger, gen *jwt.JwtService) *PostHandler {
	return &PostHandler{r: r, logger: logger, jwtGen: gen}
}

func (h *PostHandler) Initialize(r *mux.Router) *mux.Router {
	r.HandleFunc("/posts", h.Add).Methods(http.MethodPost)
	r.HandleFunc("/posts/{category}", h.List).Methods(http.MethodGet)
	r.HandleFunc("/posts/", h.List).Methods(http.MethodGet)

	r.HandleFunc("/post/{id}", h.Get).Methods(http.MethodGet)
	r.HandleFunc("/post/{id}", h.Comment).Methods(http.MethodPost)
	r.HandleFunc("/post/{id}/{comment_id}", h.DeleteComment).Methods(http.MethodDelete)

	r.HandleFunc("/post/{id}/upvote", h.Upvote).Methods(http.MethodGet)
	r.HandleFunc("/post/{id}/downvote", h.Downvote).Methods(http.MethodGet)
	r.HandleFunc("/post/{id}/unvote", h.Unvote).Methods(http.MethodGet)

	r.HandleFunc("/post/{id}", h.Delete).Methods(http.MethodDelete)
	r.HandleFunc("/user/{login}", h.User).Methods(http.MethodGet)

	return r
}

func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	posts := h.r.List(vars["category"])

	writeJSON(w, http.StatusOK, posts)
}

func (h *PostHandler) Add(w http.ResponseWriter, r *http.Request) {
	post := models.Post{}

	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Error(
			"json decoding error",
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)
	result, err := h.r.Create(post.Title, post.Category, post.Type, post.Text, post.Url, &claims.User)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("create error %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) {
	post, err := h.r.Get(mux.Vars(r)["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Comment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	comm := dto.CommentRequest{}
	err = json.NewDecoder(r.Body).Decode(&comm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Error(
			fmt.Sprintf("decoder error %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	post.AddComment(comm.Comment, &claims.User)

	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("save error %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = post.DeleteComment(vars["comment_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.logger.Error(
			fmt.Sprintf("error while deleting comment %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("error while saving post %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	post.Vote(&claims.User, 1)
	post.RecalculateScore()
	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("error while saving post %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Downvote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	post.Vote(&claims.User, -1)
	post.RecalculateScore()
	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("error while saving post %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	err = post.Unvote(&claims.User)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	post.RecalculateScore()
	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("error while saving post %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = h.r.Delete(post.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("error while deleting post %v", err),
			zap.String("request_id", w.Header().Get("X-Request-ID")))
		return
	}

	value := map[string]string{}
	value["message"] = "success"

	writeJSON(w, http.StatusOK, value)
}

func (h *PostHandler) User(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars["login"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result, err := h.r.ListByUser(vars["login"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

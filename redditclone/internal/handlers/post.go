package handlers

import (
	"encoding/json"
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

func NewPostHandler(r repository.PostRepository, gen *jwt.JwtService) *PostHandler {
	return &PostHandler{r: r, jwtGen: gen}
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
	logger := Logger(r.Context())

	post := models.Post{}

	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Warn(
			"json decode:",
			zap.Error(err),
		)
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)
	result, err := h.r.Create(post.Title, post.Category, post.Type, post.Text, post.Url, &claims.User)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post create:",
			zap.Error(err),
		)
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
	logger := Logger(r.Context())

	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	comm := dto.CommentRequest{}
	err = json.NewDecoder(r.Body).Decode(&comm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error(
			"json decode:",
			zap.Error(err))
		return
	}

	post.AddComment(comm.Comment, &claims.User)

	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	logger := Logger(r.Context())

	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))
		return
	}

	err = post.DeleteComment(vars["comment_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error(
			"comment delete:",
			zap.Error(err))
		return
	}

	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	logger := Logger(r.Context())
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	post.Vote(&claims.User, 1)
	post.RecalculateScore()
	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Downvote(w http.ResponseWriter, r *http.Request) {
	logger := Logger(r.Context())
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	post.Vote(&claims.User, -1)
	post.RecalculateScore()
	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	logger := Logger(r.Context())
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"post get:",
			zap.Error(err))
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	err = post.Unvote(&claims.User)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error(
			"unvote get:",
			zap.Error(err))
		return
	}

	post.RecalculateScore()
	err = h.r.Save(post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	logger := Logger(r.Context())
	vars := mux.Vars(r)
	post, err := h.r.Get(vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"post get:",
			zap.Error(err))
		return
	}

	err = h.r.Delete(post.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post delete:",
			zap.Error(err))
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

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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("github.com/Mimist-Illusionard/obslab/internal/handlers")

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

	ctx := r.Context()
	ctx, span := tracer.Start(r.Context(), "PostHandler.List")
	defer span.End()

	posts := h.r.List(ctx, vars["category"])

	writeJSON(w, http.StatusOK, posts)
}

func (h *PostHandler) Add(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Add")
	defer span.End()

	logger := Logger(r.Context())

	post := models.Post{}

	err := json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		logger.Warn(
			"json decode:",
			zap.Error(err),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		http.Error(w, "invalid json", http.StatusBadRequest)

		return
	}

	span.SetAttributes(attribute.String("post.title", post.Title))

	claims := r.Context().Value("claims").(jwt.Claims)
	result, err := h.r.Create(ctx, post.Title, post.Category, post.Type, post.Text, post.Url, &claims.User)
	if err != nil {
		logger.Error(
			"post create:",
			zap.Error(err),
		)

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		http.Error(w, "Error when creating post", http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Get")
	defer span.End()

	id := mux.Vars(r)["id"]
	post, err := h.r.Get(ctx, id)
	span.SetAttributes(attribute.String("post.id", id))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Comment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Comment")
	defer span.End()

	logger := Logger(r.Context())

	id := mux.Vars(r)["id"]
	post, err := h.r.Get(ctx, id)
	span.SetAttributes(attribute.String("post.id", id))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"post get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	span.SetAttributes(attribute.String("claims", fmt.Sprintf("%v", claims)))

	comm := dto.CommentRequest{}
	err = json.NewDecoder(r.Body).Decode(&comm)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error(
			"json decode:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	post.AddComment(comm.Comment, &claims.User)

	err = h.r.Save(ctx, post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.DeleteComment")
	defer span.End()

	logger := Logger(r.Context())

	vars := mux.Vars(r)
	id := vars["id"]

	span.SetAttributes(attribute.String("post id", id))

	post, err := h.r.Get(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	commentId := vars["comment_id"]
	span.SetAttributes(attribute.String("comment id", commentId))

	err = post.DeleteComment(vars["comment_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error(
			"comment delete:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	err = h.r.Save(ctx, post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Upvote")
	defer span.End()

	logger := Logger(ctx)
	vars := mux.Vars(r)

	span.SetAttributes(attribute.String("post id", vars["id"]))
	post, err := h.r.Get(ctx, vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)
	span.SetAttributes(attribute.String("claims", fmt.Sprintf("%v", claims)))

	post.Vote(&claims.User, 1)
	post.RecalculateScore()

	err = h.r.Save(ctx, post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Downvote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Downvote")
	defer span.End()

	logger := Logger(ctx)
	vars := mux.Vars(r)
	id := vars["id"]

	span.SetAttributes(attribute.String("post id", id))
	post, err := h.r.Get(ctx, id)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"comment get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)

	post.Vote(&claims.User, -1)
	post.RecalculateScore()
	err = h.r.Save(ctx, post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Unvote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Unvote")
	defer span.End()

	logger := Logger(ctx)
	vars := mux.Vars(r)

	id := vars["id"]
	span.SetAttributes(attribute.String("post.id", id))

	post, err := h.r.Get(ctx, vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"post get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	claims := r.Context().Value("claims").(jwt.Claims)
	span.SetAttributes(attribute.String("claims", fmt.Sprintf("%v", claims)))

	err = post.Unvote(&claims.User)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logger.Error(
			"unvote get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	post.RecalculateScore()

	err = h.r.Save(ctx, post)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post save:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, post)
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.Delete")
	defer span.End()

	logger := Logger(ctx)
	vars := mux.Vars(r)
	span.SetAttributes(attribute.String("post.id", vars["id"]))

	post, err := h.r.Get(ctx, vars["id"])

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		logger.Error(
			"post get:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	err = h.r.Delete(ctx, post.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		logger.Error(
			"post delete:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	value := map[string]string{}
	value["message"] = "success"

	writeJSON(w, http.StatusOK, value)
}

func (h *PostHandler) User(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "PostHandler.User")
	defer span.End()

	vars := mux.Vars(r)
	login := vars["login"]

	span.SetAttributes(attribute.String("login", login))

	if vars["login"] == "" {
		w.WriteHeader(http.StatusBadRequest)
		span.SetStatus(codes.Error, "login required")
		return
	}

	result, err := h.r.ListByUser(ctx, vars["login"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

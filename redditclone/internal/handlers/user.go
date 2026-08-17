package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Mimist-Illusionard/obslab/internal/dto"
	"github.com/Mimist-Illusionard/obslab/internal/jwt"
	"github.com/Mimist-Illusionard/obslab/internal/repository"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type UserHandler struct {
	ur     repository.UserRepository
	jwtGen *jwt.JwtService
}

func NewUserHandler(ur repository.UserRepository, gen *jwt.JwtService) *UserHandler {
	return &UserHandler{ur: ur, jwtGen: gen}
}

func (h *UserHandler) Initialize(r *mux.Router) *mux.Router {
	r.HandleFunc("/register", h.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", h.Login).Methods(http.MethodPost)

	return r
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "UserHandler.Login")
	defer span.End()

	logger := Logger(ctx)
	form := map[string]string{
		"username": "",
		"password": "",
	}

	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Error(
			"json unmarhsal:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	span.SetAttributes(attribute.String("username", form["username"]))

	user, err := h.ur.Login(ctx, form["username"], form["password"])
	if errors.Is(err, repository.ErrNoUser) {
		http.Error(w, "message:"+repository.ErrNoUser.Error(), 500)

		logger.Warn(
			"user get:",
			zap.String("username", form["username"]),
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	if err != nil {
		http.Error(w, "internal error", 500)

		logger.Error(
			"unexpected error:",
			zap.String("username", form["username"]),
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	token, err := h.jwtGen.GenerateJwt(user)

	if err != nil {
		http.Error(w, "error generating token", 500)

		logger.Error(
			"token generate::",
			zap.String("username", form["username"]),
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tracer.Start(ctx, "UserHandler.Register")
	defer span.End()

	logger := Logger(ctx)
	form := map[string]string{
		"username": "",
		"password": "",
	}

	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.Error(
			"json unmarhsal:",
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	span.SetAttributes(attribute.String("username", form["username"]))

	user, err := h.ur.Register(ctx, form["username"], form["password"])
	if err != nil && errors.Is(err, repository.ErrHasUser) {

		errResponse := dto.ErrorResponse{
			Errors: []dto.ValidationError{
				{
					Location: "body",
					Param:    "username",
					Value:    form["username"],
					Msg:      err.Error(),
				},
			},
		}

		logger.Warn(
			"user exist:",
			zap.String("username", form["username"]),
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)

		json.NewEncoder(w).Encode(errResponse)
		return
	}

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		logger.Warn(
			"unexpected error:",
			zap.String("username", form["username"]),
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	token, err := h.jwtGen.GenerateJwt(user)

	if err != nil {
		http.Error(w, "error generating token", http.StatusInternalServerError)
		logger.Warn(
			"token generate:",
			zap.String("username", form["username"]),
			zap.Error(err))

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

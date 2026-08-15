package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Mimist-Illusionard/obslab/internal/dto"
	"github.com/Mimist-Illusionard/obslab/internal/jwt"
	"github.com/Mimist-Illusionard/obslab/internal/repository"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type UserHandler struct {
	ur     repository.UserRepository
	jwtGen *jwt.JwtService
	logger *zap.Logger
}

func NewUserHandler(ur repository.UserRepository, logger *zap.Logger, gen *jwt.JwtService) *UserHandler {
	return &UserHandler{ur: ur, logger: logger, jwtGen: gen}
}

func (h *UserHandler) Initialize(r *mux.Router) *mux.Router {
	r.HandleFunc("/register", h.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", h.Login).Methods(http.MethodPost)

	return r
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	form := map[string]string{
		"username": "",
		"password": "",
	}

	requestID := w.Header().Get("X-Request-ID")
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error(
			fmt.Sprintf("json unmarhsal: %v", err.Error()),
			zap.String("request_id", requestID))
		return
	}

	user, err := h.ur.Login(form["username"], form["password"])
	if errors.Is(err, repository.ErrNoUser) {
		http.Error(w, "message:"+repository.ErrNoUser.Error(), 500)

		h.logger.Warn(
			fmt.Sprintf("user get: %s", err.Error()),
			zap.String("request_id", requestID),
			zap.String("username", form["username"]))

		return
	}
	if err != nil {
		http.Error(w, "internal error", 500)

		h.logger.Error(
			fmt.Sprintf("unexpected error: %v", err.Error()),
			zap.String("request_id", requestID),
			zap.String("username", form["username"]))

		return
	}

	token, err := h.jwtGen.GenerateJwt(user)

	if err != nil {
		http.Error(w, "error generating token", 500)
		h.logger.Error(
			fmt.Sprintf("token generate: %v", err.Error()),
			zap.String("request_id", requestID),
			zap.String("username", form["username"]))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	form := map[string]string{
		"username": "",
		"password": "",
	}

	requestID := w.Header().Get("X-Request-ID")
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		h.logger.Error(
			fmt.Sprintf("json unmarshal: %v", err.Error()),
			zap.String("request_id", requestID))
		return
	}

	user, err := h.ur.Register(form["username"], form["password"])
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

		h.logger.Warn(
			fmt.Sprintf("user exist: %v", err.Error()),
			zap.String("request_id", requestID),
			zap.String("username", form["username"]))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)

		json.NewEncoder(w).Encode(errResponse)
		return
	}

	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		h.logger.Error(
			fmt.Sprintf("unexpected error: %v", err.Error()),
			zap.String("request_id", requestID),
			zap.String("username", form["username"]))
		return
	}

	token, err := h.jwtGen.GenerateJwt(user)

	if err != nil {
		http.Error(w, "error generating token", http.StatusInternalServerError)
		h.logger.Error(
			fmt.Sprintf("token generate: %v", err.Error()),
			zap.String("request_id", requestID),
			zap.String("username", form["username"]))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

package http_server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	httpPack "github.com/kingxl111/mosprom/AuthService/internal/environment"
	models "github.com/kingxl111/mosprom/AuthService/internal/user"
	api "github.com/kingxl111/mosprom/AuthService/pkg/api/auth"
	"github.com/oapi-codegen/runtime/types"
)

var _ api.ServerInterface = (*Handler)(nil)

type Handler struct {
	svc    AuthService
	logger *slog.Logger
}

func NewHandler(svc AuthService, logger *slog.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

func (h *Handler) PostApiV1AuthRegister(w http.ResponseWriter, r *http.Request) {
	var req api.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}

	// Валидация входных данных
	if req.Email == "" || req.Password == "" || req.Role == "" {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("email, password and role are required"))
		return
	}

	if len(req.Password) < 8 {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("password must be at least 8 characters"))
		return
	}

	validRoles := map[string]bool{"company": true, "admin": true, "analyst": true}
	if !validRoles[string(req.Role)] {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid role"))
		return
	}

	resp, err := h.svc.Register(r.Context(), &models.RegisterRequest{
		Email:    string(req.Email),
		Password: req.Password,
		Role:     string(req.Role),
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, api.AuthResponse{
		AccessToken:  &resp.AccessToken,
		RefreshToken: &resp.RefreshToken,
	})
}

func (h *Handler) PostApiV1AuthLogin(w http.ResponseWriter, r *http.Request) {
	var req api.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}

	// Валидация входных данных
	if req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("email and password are required"))
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	resp, err := h.svc.Login(r.Context(), &models.LoginRequest{
		Email:     string(req.Email),
		Password:  req.Password,
		IP:        ip,
		UserAgent: userAgent,
	})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, api.AuthResponse{
		AccessToken:  &resp.AccessToken,
		RefreshToken: &resp.RefreshToken,
	})
}

func (h *Handler) PostApiV1AuthRefresh(w http.ResponseWriter, r *http.Request) {
	var req api.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, api.AuthResponse{
		AccessToken: &resp.AccessToken,
	})
}

func (h *Handler) PostApiV1AuthLogout(w http.ResponseWriter, r *http.Request) {
	type logoutReq struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req logoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) GetApiV1UsersMe(w http.ResponseWriter, r *http.Request) {
	uidVal := r.Context().Value(httpPack.ContextUserIDKey)
	if uidVal == nil {
		h.writeError(w, http.StatusUnauthorized, fmt.Errorf("missing user id in context"))
		return
	}

	// contexte stores int
	userID, ok := uidVal.(int)
	if !ok {
		// maybe it was stored as float64 (json) — try convert if needed
		// but better to ensure middleware stores int
		h.writeError(w, http.StatusInternalServerError, fmt.Errorf("invalid user id type in context"))
		return
	}

	userResp, err := h.svc.GetUserByID(r.Context(), userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	email := types.Email(userResp.Email)
	role := api.UserInfoResponseRole(userResp.Role)
	resp := api.UserInfoResponse{
		Id:        strPtr(fmt.Sprintf("%d", userResp.ID)),
		Email:     &email,
		Role:      &role,
		CreatedAt: &userResp.CreatedAt,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	h.logger.Error("service error", "err", err)

	switch {
	case errors.Is(err, models.ErrInvalidCredentials):
		h.writeError(w, http.StatusUnauthorized, err)
	case errors.Is(err, models.ErrUserExists):
		h.writeError(w, http.StatusConflict, err)
	case errors.Is(err, models.ErrTokenExpired):
		h.writeError(w, http.StatusUnauthorized, err)
	default:
		h.writeError(w, http.StatusInternalServerError, err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, err error) {
	msg := err.Error()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := api.ErrorResponse{Error: &msg}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("encode json", "err", err)
	}
}

func strPtr(s string) *string {
	return &s
}

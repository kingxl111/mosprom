package http_server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	httpPack "github.com/kingxl111/mosprom/AuthService/internal/environment"
	"github.com/kingxl111/mosprom/AuthService/internal/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAuthService is a simple mock for testing
type MockAuthService struct {
	registerFunc func(ctx context.Context, req *user.RegisterRequest) (*user.AuthResponse, error)
	loginFunc    func(ctx context.Context, req *user.LoginRequest) (*user.AuthResponse, error)
	refreshFunc  func(ctx context.Context, token string) (*user.TokenResponse, error)
	logoutFunc   func(ctx context.Context, token string) error
	getUserFunc  func(ctx context.Context, id int) (*user.UserResponse, error)
}

func (m *MockAuthService) Register(ctx context.Context, req *user.RegisterRequest) (*user.AuthResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockAuthService) Login(ctx context.Context, req *user.LoginRequest) (*user.AuthResponse, error) {
	if m.loginFunc != nil {
		return m.loginFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockAuthService) Refresh(ctx context.Context, token string) (*user.TokenResponse, error) {
	if m.refreshFunc != nil {
		return m.refreshFunc(ctx, token)
	}
	return nil, nil
}

func (m *MockAuthService) Logout(ctx context.Context, token string) error {
	if m.logoutFunc != nil {
		return m.logoutFunc(ctx, token)
	}
	return nil
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id int) (*user.UserResponse, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, id)
	}
	return nil, nil
}

func TestHandler_PostApiV1AuthRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful registration",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"role":     "company",
			},
			mockSetup: func(m *MockAuthService) {
				m.registerFunc = func(ctx context.Context, req *user.RegisterRequest) (*user.AuthResponse, error) {
					return &user.AuthResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"password": "password123",
				"role":     "company",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email, password and role are required",
		},
		{
			name: "short password",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "123",
				"role":     "company",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "password must be at least 8 characters",
		},
		{
			name: "invalid role",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
				"role":     "invalid",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid role",
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewHandler(mockService, slog.Default())

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PostApiV1AuthRegister(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResp struct {
					Error *string `json:"error"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.NotNil(t, errorResp.Error)
				assert.Contains(t, *errorResp.Error, tt.expectedError)
			} else if tt.expectedStatus == http.StatusOK {
				var resp struct {
					AccessToken  *string `json:"access_token"`
					RefreshToken *string `json:"refresh_token"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotNil(t, resp.AccessToken)
				assert.NotNil(t, resp.RefreshToken)
			}
		})
	}
}

func TestHandler_PostApiV1AuthLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful login",
			requestBody: map[string]interface{}{
				"email":    "test@example.com",
				"password": "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.loginFunc = func(ctx context.Context, req *user.LoginRequest) (*user.AuthResponse, error) {
					return &user.AuthResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing email",
			requestBody: map[string]interface{}{
				"password": "password123",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email and password are required",
		},
		{
			name: "missing password",
			requestBody: map[string]interface{}{
				"email": "test@example.com",
			},
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "email and password are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewHandler(mockService, slog.Default())

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.requestBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.PostApiV1AuthLogin(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResp struct {
					Error *string `json:"error"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.NotNil(t, errorResp.Error)
				assert.Contains(t, *errorResp.Error, tt.expectedError)
			} else if tt.expectedStatus == http.StatusOK {
				var resp struct {
					AccessToken  *string `json:"access_token"`
					RefreshToken *string `json:"refresh_token"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotNil(t, resp.AccessToken)
				assert.NotNil(t, resp.RefreshToken)
			}
		})
	}
}

func TestHandler_GetApiV1UsersMe(t *testing.T) {
	tests := []struct {
		name           string
		contextSetup   func(*http.Request) *http.Request
		mockSetup      func(*MockAuthService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful get user",
			contextSetup: func(req *http.Request) *http.Request {
				ctx := context.WithValue(req.Context(), httpPack.ContextUserIDKey, 1)
				return req.WithContext(ctx)
			},
			mockSetup: func(m *MockAuthService) {
				m.getUserFunc = func(ctx context.Context, id int) (*user.UserResponse, error) {
					return &user.UserResponse{
						ID:    1,
						Email: "test@example.com",
						Role:  "company",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user id in context",
			contextSetup:   func(req *http.Request) *http.Request { return req },
			mockSetup:      func(m *MockAuthService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing user id in context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockAuthService{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			handler := NewHandler(mockService, slog.Default())

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			if tt.contextSetup != nil {
				req = tt.contextSetup(req)
			}
			w := httptest.NewRecorder()

			handler.GetApiV1UsersMe(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResp struct {
					Error *string `json:"error"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.NotNil(t, errorResp.Error)
				assert.Contains(t, *errorResp.Error, tt.expectedError)
			} else if tt.expectedStatus == http.StatusOK {
				var resp struct {
					ID    *string `json:"id"`
					Email *string `json:"email"`
					Role  *string `json:"role"`
				}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.NotNil(t, resp.ID)
				assert.NotNil(t, resp.Email)
				assert.NotNil(t, resp.Role)
			}
		})
	}
}

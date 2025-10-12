package proxy

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kingxl111/mosprom/ApiGateway/internal/config"
)

type AuthServiceProxy interface {
	ProxyHandler() http.Handler
}

type authServiceProxy struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func NewAuthServiceProxy(cfg config.AuthServiceConfig, logger *slog.Logger) AuthServiceProxy {
	return &authServiceProxy{
		baseURL: cfg.BaseURL(),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (p *authServiceProxy) ProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Создаем новый запрос к AuthService
		proxyURL := p.baseURL + r.URL.Path
		if r.URL.RawQuery != "" {
			proxyURL += "?" + r.URL.RawQuery
		}

		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			p.logger.Error("failed to read request body", slog.Any("err", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Создаем новый запрос
		proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, proxyURL, bytes.NewReader(body))
		if err != nil {
			p.logger.Error("failed to create proxy request", slog.Any("err", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Копируем заголовки
		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Выполняем запрос
		resp, err := p.client.Do(proxyReq)
		if err != nil {
			p.logger.Error("failed to execute proxy request", slog.Any("err", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Копируем заголовки ответа
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Устанавливаем статус код
		w.WriteHeader(resp.StatusCode)

		// Копируем тело ответа
		if _, err := io.Copy(w, resp.Body); err != nil {
			p.logger.Error("failed to copy response body", slog.Any("err", err))
		}
	})
}

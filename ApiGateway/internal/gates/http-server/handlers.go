package http_server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kingxl111/mosprom/ApiGateway/internal/redis"
)

type Handler struct {
	redisClient redis.Client
	logger      *slog.Logger
}

func NewHandler(authProxy interface{}, redisClient redis.Client, logger *slog.Logger) *Handler {
	return &Handler{
		redisClient: redisClient,
		logger:      logger,
	}
}

type FileUploadRequest struct {
	File io.Reader `json:"-"`
	Name string    `json:"name"`
	Type string    `json:"type"`
}

type FileUploadResponse struct {
	FileID    string `json:"file_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type FileStatusResponse struct {
	FileID    string `json:"file_id"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Message   string `json:"message"`
	ResultURL string `json:"result_url,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		h.logger.Error("failed to parse multipart form", slog.Any("err", err))
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.logger.Error("failed to get file from form", slog.Any("err", err))
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Генерируем уникальный ID для файла
	fileID := uuid.New().String()

	// Сохраняем информацию о файле в Redis
	fileInfo := map[string]interface{}{
		"file_id":    fileID,
		"filename":   header.Filename,
		"size":       header.Size,
		"status":     "uploaded",
		"created_at": time.Now().Format(time.RFC3339),
		"updated_at": time.Now().Format(time.RFC3339),
	}

	fileInfoJSON, err := json.Marshal(fileInfo)
	if err != nil {
		h.logger.Error("failed to marshal file info", slog.Any("err", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Сохраняем в Redis с TTL 24 часа
	ctx := r.Context()
	err = h.redisClient.Publish(ctx, "file_uploaded", string(fileInfoJSON))
	if err != nil {
		h.logger.Error("failed to publish file upload event", slog.Any("err", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Отправляем файл в Kafka для обработки
	// Здесь должен быть код для отправки в Kafka
	// Пока что просто логируем
	h.logger.Info("file uploaded",
		slog.String("file_id", fileID),
		slog.String("filename", header.Filename),
		slog.Int64("size", header.Size))

	// Отвечаем клиенту
	response := FileUploadResponse{
		FileID:    fileID,
		Status:    "uploaded",
		Message:   "File uploaded successfully and queued for processing",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetFileStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем file_id из URL
	fileID := filepath.Base(r.URL.Path)
	if fileID == "" {
		http.Error(w, "File ID is required", http.StatusBadRequest)
		return
	}

	// Здесь должен быть код для получения статуса из Redis
	// Пока что возвращаем заглушку
	response := FileStatusResponse{
		FileID:    fileID,
		Status:    "processing",
		Progress:  50,
		Message:   "File is being processed",
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

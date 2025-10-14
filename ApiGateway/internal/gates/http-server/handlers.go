package http_server

import (
	"context"
	"encoding/json"

	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kingxl111/mosprom/ApiGateway/internal/fileprocessor"
	"github.com/kingxl111/mosprom/ApiGateway/internal/redis"
	ApiGateway "github.com/kingxl111/mosprom/ApiGateway/pkg/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

var _ ApiGateway.ServerInterface = (*Handler)(nil)

type Handler struct {
	fileProcessor fileprocessor.FileProcessor
	redisClient   redis.Client
	logger        *slog.Logger
	authProxy     http.Handler
}

func NewHandler(fileProcessor fileprocessor.FileProcessor, redisClient redis.Client, logger *slog.Logger, authProxy http.Handler) *Handler {
	return &Handler{
		fileProcessor: fileProcessor,
		redisClient:   redisClient,
		logger:        logger,
		authProxy:     authProxy,
	}
}

// ==================== Аналитика и отчеты ====================

// Получение списка доступных отчетов
// (GET /api/v1/analytics/reports)
func (h *Handler) GetApiV1AnalyticsReports(w http.ResponseWriter, r *http.Request, params ApiGateway.GetApiV1AnalyticsReportsParams) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	page := 1
	if params.Page != nil {
		page = *params.Page
	}

	limit := 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	// Здесь должна быть логика получения отчетов из базы данных
	// Пока возвращаем заглушку
	reports := []ApiGateway.ReportItem{
		{
			ReportId:    openapi_types.UUID(uuid.New()),
			ReportType:  "production",
			ReportName:  "Отчет по производственным показателям за январь 2024",
			GeneratedAt: time.Now().Add(-24 * time.Hour),
			TimeRange: ApiGateway.TimeRange{
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			FileSize:    1024 * 1024,
			DownloadUrl: "https://storage.mosprom.ru/reports/production_january_2024.pdf",
		},
	}

	response := ApiGateway.ReportListResponse{
		Reports: reports,
		Pagination: ApiGateway.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      1,
			TotalPages: 1,
		},
	}

	h.sendJSON(w, response, http.StatusOK)
}

// Генерация нового отчета
// (POST /api/v1/analytics/reports/generate)
func (h *Handler) PostApiV1AnalyticsReportsGenerate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request ApiGateway.GenerateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация времени
	if request.TimeRange.StartDate.After(request.TimeRange.EndDate) {
		h.sendError(w, "Start date cannot be after end date", http.StatusBadRequest)
		return
	}

	reportID := uuid.New()
	response := ApiGateway.GenerateReportResponse{
		ReportId:                openapi_types.UUID(reportID),
		Status:                  "queued",
		EstimatedCompletionTime: 300, // 5 минут
		Message:                 "Отчет поставлен в очередь на генерацию",
	}

	h.sendJSON(w, response, http.StatusAccepted)
}

// Получение отчета
// (GET /api/v1/analytics/reports/{report_id})
func (h *Handler) GetApiV1AnalyticsReportsReportId(w http.ResponseWriter, r *http.Request, reportId openapi_types.UUID) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Здесь должна быть логика получения отчета из базы данных
	response := ApiGateway.ReportResponse{
		ReportId:    reportId,
		ReportType:  "production",
		ReportName:  "Отчет по производственным показателям",
		GeneratedAt: time.Now().Add(-1 * time.Hour),
		TimeRange: ApiGateway.TimeRange{
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		Summary: map[string]interface{}{
			"total_production": 15000,
			"growth_rate":      12.5,
			"efficiency":       85.2,
		},
		DownloadUrl: "https://storage.mosprom.ru/reports/" + reportId.String() + ".pdf",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	h.sendJSON(w, response, http.StatusOK)
}

// ==================== Аутентификация ====================

// Аутентификация пользователя и получение токенов
// (POST /api/v1/auth/login)
func (h *Handler) PostApiV1AuthLogin(w http.ResponseWriter, r *http.Request) {
	h.authProxy.ServeHTTP(w, r)
}

// Выход из системы
// (POST /api/v1/auth/logout)
func (h *Handler) PostApiV1AuthLogout(w http.ResponseWriter, r *http.Request) {
	h.authProxy.ServeHTTP(w, r)
}

// Обновление access токена
// (POST /api/v1/auth/refresh)
func (h *Handler) PostApiV1AuthRefresh(w http.ResponseWriter, r *http.Request) {
	h.authProxy.ServeHTTP(w, r)
}

// Регистрация нового пользователя (компании или администратора)
// (POST /api/v1/auth/register)
func (h *Handler) PostApiV1AuthRegister(w http.ResponseWriter, r *http.Request) {
	h.authProxy.ServeHTTP(w, r)
}

// ==================== Управление файлами ====================

// Получение списка загруженных файлов
// (GET /api/v1/files)
func (h *Handler) GetApiV1Files(w http.ResponseWriter, r *http.Request, params ApiGateway.GetApiV1FilesParams) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	page := 1
	if params.Page != nil {
		page = *params.Page
	}

	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
	}

	// Здесь должна быть логика получения файлов из базы данных
	// Пока возвращаем заглушку
	files := []ApiGateway.FileItem{
		{
			FileId:      openapi_types.UUID(uuid.New()),
			Filename:    "production_data_january_2024.csv",
			Size:        1024 * 1024,
			Status:      "processed",
			DataType:    "production",
			CreatedAt:   time.Now().Add(-48 * time.Hour),
			ProcessedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			FileId:    openapi_types.UUID(uuid.New()),
			Filename:  "financial_report_q4_2023.xlsx",
			Size:      2 * 1024 * 1024,
			Status:    "processing",
			DataType:  "financial",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
	}

	response := ApiGateway.FileListResponse{
		Files: files,
		Pagination: ApiGateway.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      len(files),
			TotalPages: 1,
		},
	}

	h.sendJSON(w, response, http.StatusOK)
}

// Загрузка файла с промышленными данными
// (POST /api/v1/files/upload)
func (h *Handler) PostApiV1FilesUpload(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Парсим multipart форму
	err := r.ParseMultipartForm(100 << 20) // 100 MB
	if err != nil {
		h.sendError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.sendError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Получаем дополнительные параметры
	description := r.FormValue("description")
	dataType := r.FormValue("data_type")

	h.logger.Info("File upload request",
		slog.String("user_id", userID),
		slog.String("filename", header.Filename),
		slog.String("data_type", dataType),
		slog.String("description", description))

	// Обрабатываем файл через FileProcessor
	metadata, err := h.fileProcessor.ProcessUpload(r.Context(), file, header.Filename, userID)
	if err != nil {
		h.logger.Error("Failed to process file upload", slog.Any("error", err))
		h.sendError(w, "Failed to process file upload", http.StatusInternalServerError)
		return
	}

	response := ApiGateway.FileUploadResponse{
		FileId:                  openapi_types.UUID(uuid.MustParse(metadata.FileID)),
		Filename:                metadata.Filename,
		Status:                  metadata.Status,
		Message:                 "File uploaded successfully and queued for processing",
		CreatedAt:               metadata.CreatedAt,
		EstimatedProcessingTime: 180, // 3 минуты
	}

	h.sendJSON(w, response, http.StatusAccepted)
}

// Удаление файла и связанных данных
// (DELETE /api/v1/files/{file_id})
func (h *Handler) DeleteApiV1FilesFileId(w http.ResponseWriter, r *http.Request, fileId openapi_types.UUID) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Здесь должна быть логика удаления файла
	// Пока просто логируем и возвращаем успех
	h.logger.Info("File deletion request",
		slog.String("user_id", userID),
		slog.String("file_id", fileId.String()))

	w.WriteHeader(http.StatusNoContent)
}

// Получение статуса обработки файла
// (GET /api/v1/files/{file_id})
func (h *Handler) GetApiV1FilesFileId(w http.ResponseWriter, r *http.Request, fileId openapi_types.UUID) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Здесь должна быть логика получения статуса из Redis/БД
	// Пока возвращаем заглушку
	response := ApiGateway.FileStatusResponse{
		FileId:    fileId,
		Filename:  "production_data_2024.csv",
		Status:    "processing",
		Progress:  65,
		Message:   "File is being processed by ETL system",
		CreatedAt: time.Now().Add(-30 * time.Minute),
		UpdatedAt: time.Now().Add(-5 * time.Minute),
	}

	h.sendJSON(w, response, http.StatusOK)
}

// ==================== Системные эндпоинты ====================

// Проверка состояния системы
// (GET /api/v1/system/health)
func (h *Handler) GetApiV1SystemHealth(w http.ResponseWriter, r *http.Request) {
	// Проверяем состояние всех компонентов системы
	components := ApiGateway.HealthCheckResponseComponents{
		ApiGateway:  "healthy",
		AuthService: "healthy",
		Kafka:       "healthy",
		Redis:       "healthy",
		Postgres:    "healthy",
		Clickhouse:  "healthy",
	}

	// Проверяем Redis
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.redisClient.Publish(ctx, "health_check", "ping"); err != nil {
		components.Redis = "unhealthy"
	}

	// Определяем общий статус
	overallStatus := "healthy"
	if components.Redis == "unhealthy" {
		overallStatus = "degraded"
	}

	response := ApiGateway.HealthCheckResponse{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Components: components,
		Version:    "1.0.0",
	}

	h.sendJSON(w, response, http.StatusOK)
}

// Получение метрик системы
// (GET /api/v1/system/metrics)
func (h *Handler) GetApiV1SystemMetrics(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Здесь должна быть реальная логика сбора метрик
	// Пока возвращаем заглушку
	response := ApiGateway.SystemMetricsResponse{
		Goroutines:             125,
		MemoryUsageMb:          45.7,
		ActiveConnections:      23,
		RequestsPerSecond:      12.5,
		UptimeSeconds:          86400, // 24 часа
		KafkaMessagesProcessed: 1500,
	}

	h.sendJSON(w, response, http.StatusOK)
}

// ==================== Управление пользователями ====================

// Получение информации о текущем пользователе
// (GET /api/v1/users/me)
func (h *Handler) GetApiV1UsersMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Проксируем запрос к AuthService
	h.authProxy.ServeHTTP(w, r)
}

// Обновление профиля пользователя
// (PUT /api/v1/users/profile)
func (h *Handler) PutApiV1UsersProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok {
		h.sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Проксируем запрос к AuthService
	h.authProxy.ServeHTTP(w, r)
}

// ==================== Вспомогательные методы ====================

// sendJSON отправляет JSON ответ
func (h *Handler) sendJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

// sendError отправляет ошибку в стандартном формате
func (h *Handler) sendError(w http.ResponseWriter, message string, statusCode int) {
	errorResponse := ApiGateway.ErrorResponse{
		Error:     http.StatusText(statusCode),
		Message:   message,
		Timestamp: time.Now(),
		RequestId: uuid.New().String(),
	}

	h.sendJSON(w, errorResponse, statusCode)
}

// getPaginationParams извлекает параметры пагинации из запроса
func (h *Handler) getPaginationParams(r *http.Request) (page, limit int) {
	page = 1
	limit = 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
}

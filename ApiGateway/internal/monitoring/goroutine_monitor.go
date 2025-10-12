package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

type GoroutineMonitor interface {
	Start(ctx context.Context) error
	GetStats() GoroutineStats
}

type GoroutineStats struct {
	CurrentGoroutines int           `json:"current_goroutines"`
	MaxGoroutines     int           `json:"max_goroutines"`
	TotalGoroutines   int64         `json:"total_goroutines"`
	LeakThreshold     int           `json:"leak_threshold"`
	LastCheck         time.Time     `json:"last_check"`
	CheckInterval     time.Duration `json:"check_interval"`
}

type goroutineMonitor struct {
	logger          *slog.Logger
	stats           GoroutineStats
	mu              sync.RWMutex
	checkInterval   time.Duration
	leakThreshold   int
	maxGoroutines   int
	totalGoroutines int64
	running         bool
	stopChan        chan struct{}
}

func NewGoroutineMonitor(logger *slog.Logger) GoroutineMonitor {
	return &goroutineMonitor{
		logger:          logger,
		checkInterval:   30 * time.Second,
		leakThreshold:   1000, // Порог для обнаружения утечек
		maxGoroutines:   0,
		totalGoroutines: 0,
		stopChan:        make(chan struct{}),
	}
}

func (m *goroutineMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("monitor is already running")
	}

	m.running = true
	m.stats = GoroutineStats{
		CheckInterval: m.checkInterval,
		LeakThreshold: m.leakThreshold,
		LastCheck:     time.Now(),
	}

	// Запускаем мониторинг в отдельной горутине
	go m.monitorLoop(ctx)

	m.logger.Info("goroutine monitor started",
		slog.Duration("check_interval", m.checkInterval),
		slog.Int("leak_threshold", m.leakThreshold))

	return nil
}

func (m *goroutineMonitor) GetStats() GoroutineStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

func (m *goroutineMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("goroutine monitor context cancelled")
			return
		case <-m.stopChan:
			m.logger.Info("goroutine monitor shutdown requested")
			return
		case <-ticker.C:
			m.checkGoroutines()
		}
	}
}

func (m *goroutineMonitor) checkGoroutines() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Получаем текущее количество горутин
	currentGoroutines := runtime.NumGoroutine()

	// Обновляем статистику
	m.stats.CurrentGoroutines = currentGoroutines
	m.stats.LastCheck = time.Now()

	// Обновляем максимум
	if currentGoroutines > m.maxGoroutines {
		m.maxGoroutines = currentGoroutines
		m.stats.MaxGoroutines = m.maxGoroutines
	}

	// Увеличиваем общий счетчик
	m.totalGoroutines++
	m.stats.TotalGoroutines = m.totalGoroutines

	// Проверяем на утечки
	if currentGoroutines > m.leakThreshold {
		m.logger.Warn("potential goroutine leak detected",
			slog.Int("current_goroutines", currentGoroutines),
			slog.Int("leak_threshold", m.leakThreshold),
			slog.Int("max_goroutines", m.maxGoroutines))
	}

	// Логируем статистику каждые 10 проверок
	if m.totalGoroutines%10 == 0 {
		m.logger.Info("goroutine statistics",
			slog.Int("current_goroutines", currentGoroutines),
			slog.Int("max_goroutines", m.maxGoroutines),
			slog.Int64("total_checks", m.totalGoroutines))
	}

	// Дополнительная диагностика при подозрении на утечку
	if currentGoroutines > m.leakThreshold/2 {
		m.logDetailedStats()
	}
}

func (m *goroutineMonitor) logDetailedStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.logger.Info("detailed goroutine statistics",
		slog.Int("current_goroutines", m.stats.CurrentGoroutines),
		slog.Int("max_goroutines", m.stats.MaxGoroutines),
		slog.Uint64("heap_alloc", memStats.HeapAlloc),
		slog.Uint64("heap_sys", memStats.HeapSys),
		slog.Int("num_gc", int(memStats.NumGC)),
		slog.Duration("gc_pause_total", time.Duration(memStats.PauseTotalNs)))
}

// Stop останавливает мониторинг
func (m *goroutineMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopChan)
	m.running = false

	m.logger.Info("goroutine monitor stopped",
		slog.Int("final_goroutines", m.stats.CurrentGoroutines),
		slog.Int("max_goroutines", m.stats.MaxGoroutines),
		slog.Int64("total_checks", m.stats.TotalGoroutines))
}

// GetGoroutineStack возвращает стек всех горутин (для отладки)
func (m *goroutineMonitor) GetGoroutineStack() []byte {
	buf := make([]byte, 1024*1024) // 1MB буфер
	n := runtime.Stack(buf, true)
	return buf[:n]
}

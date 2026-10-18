package health

import (
	"context"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Checker defines an interface for dependency health checks.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// Handler provides health check endpoints.
type Handler struct {
	checkers []Checker
	mu       sync.RWMutex
}

// NewHandler creates a new health handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Register adds a health checker.
func (h *Handler) Register(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checker)
}

// Health returns a simple OK response.
func (h *Handler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Live returns liveness probe — always 200 if process is running.
func (h *Handler) Live(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "alive"})
}

// Ready checks all registered dependencies and returns readiness status.
func (h *Handler) Ready(c *fiber.Ctx) error {
	h.mu.RLock()
	checkers := make([]Checker, len(h.checkers))
	copy(checkers, h.checkers)
	h.mu.RUnlock()

	ctx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	var (
		wg sync.WaitGroup
	)
	results := make(map[string]string, len(checkers))
	var resultsMu sync.Mutex
	allOK := true

	for _, checker := range checkers {
		wg.Add(1)
		go func(ch Checker) {
			defer wg.Done()
			if err := ch.Check(ctx); err != nil {
				resultsMu.Lock()
				results[ch.Name()] = "unavailable: " + err.Error()
				resultsMu.Unlock()
				allOK = false
			} else {
				resultsMu.Lock()
				results[ch.Name()] = "available"
				resultsMu.Unlock()
			}
		}(checker)
	}
	wg.Wait()

	if !allOK {
		return c.Status(503).JSON(fiber.Map{
			"status": "unavailable",
			"checks": results,
		})
	}

	return c.JSON(fiber.Map{
		"status": "ready",
		"checks": results,
	})
}

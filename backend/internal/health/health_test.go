package health

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

type mockChecker struct {
	name string
	err  error
}

func (m *mockChecker) Name() string                  { return m.name }
func (m *mockChecker) Check(_ context.Context) error { return m.err }

func TestHealth(t *testing.T) {
	h := NewHandler()
	app := fiber.New()
	app.Get("/health", h.Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestLive(t *testing.T) {
	h := NewHandler()
	app := fiber.New()
	app.Get("/health/live", h.Live)

	req, _ := http.NewRequest("GET", "/health/live", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestReady_AllHealthy(t *testing.T) {
	h := NewHandler()
	h.Register(&mockChecker{name: "mongo", err: nil})
	h.Register(&mockChecker{name: "nats", err: nil})

	app := fiber.New()
	app.Get("/health/ready", h.Ready)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestReady_Unhealthy(t *testing.T) {
	h := NewHandler()
	h.Register(&mockChecker{name: "mongo", err: nil})
	h.Register(&mockChecker{name: "nats", err: fmt.Errorf("connection refused")})

	app := fiber.New()
	app.Get("/health/ready", h.Ready)

	req, _ := http.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)
}

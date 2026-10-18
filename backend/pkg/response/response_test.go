package response

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestOK(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return OK(c, map[string]string{"key": "value"})
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.True(t, result["success"].(bool))
}

func TestCreated(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		return Created(c, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest("POST", "/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestError_AppError(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return Error(c, errors.ErrNotFound)
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	assert.False(t, result["success"].(bool))
	errObj := result["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errObj["code"])
}

func TestError_GenericError(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return Error(c, errors.New("CUSTOM", "custom error", 422))
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 422, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	errObj := result["error"].(map[string]interface{})
	assert.Equal(t, "custom error", errObj["message"])
}

func TestPaginatedMeta(t *testing.T) {
	meta := PaginatedMeta(1, 10, 25)
	assert.Equal(t, 1, meta.Page)
	assert.Equal(t, 10, meta.PerPage)
	assert.Equal(t, int64(25), meta.Total)
	assert.Equal(t, 3, meta.TotalPages)

	meta = PaginatedMeta(1, 10, 30)
	assert.Equal(t, 3, meta.TotalPages)

	meta = PaginatedMeta(1, 10, 0)
	assert.Equal(t, 0, meta.TotalPages)
}

func TestOKWithMeta_RequestID(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		c.Locals("request_id", "req-123")
		return OKWithMeta(c, "data", &Meta{Page: 1, PerPage: 10, Total: 100, TotalPages: 10})
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	meta := result["meta"].(map[string]interface{})
	assert.Equal(t, "req-123", meta["request_id"])
}

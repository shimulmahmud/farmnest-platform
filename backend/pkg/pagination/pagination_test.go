package pagination

import (
	"net/http/httptest"
	"testing"

	"github.com/ecommerce/backend/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestParse_Defaults(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return response.Error(c, err)
		},
	})
	app.Get("/", func(c *fiber.Ctx) error {
		p, err := Parse(c)
		if err != nil {
			return err
		}
		return c.JSON(p)
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestParse_CustomValues(t *testing.T) {
	var p Params
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		var err error
		p, err = Parse(c)
		return err
	})

	req := httptest.NewRequest("GET", "/?page=3&per_page=50", nil)
	_, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 3, p.Page)
	assert.Equal(t, 50, p.PerPage)
	assert.Equal(t, 100, p.Skip) // (3-1)*50
	assert.Equal(t, 50, p.Limit)
}

func TestParse_InvalidPage(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return response.Error(c, err)
		},
	})
	app.Get("/", func(c *fiber.Ctx) error {
		_, err := Parse(c)
		return err
	})

	req := httptest.NewRequest("GET", "/?page=abc", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 422, resp.StatusCode)
}

func TestParse_InvalidPerPage(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return response.Error(c, err)
		},
	})
	app.Get("/", func(c *fiber.Ctx) error {
		_, err := Parse(c)
		return err
	})

	req := httptest.NewRequest("GET", "/?per_page=-1", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 422, resp.StatusCode)
}

func TestParse_MaxPerPage(t *testing.T) {
	var p Params
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		var err error
		p, err = Parse(c)
		return err
	})

	req := httptest.NewRequest("GET", "/?per_page=500", nil)
	_, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, MaxPerPage, p.PerPage)
	assert.Equal(t, MaxPerPage, p.Limit)
}

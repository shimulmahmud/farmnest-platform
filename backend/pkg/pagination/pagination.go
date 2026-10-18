package pagination

import (
	"strconv"

	"github.com/ecommerce/backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Params holds parsed pagination parameters.
type Params struct {
	Page    int
	PerPage int
	Skip    int
	Limit   int
}

// Parse extracts and validates pagination params from query string.
func Parse(c *fiber.Ctx) (Params, error) {
	page, err := strconv.Atoi(c.Query("page", strconv.Itoa(DefaultPage)))
	if err != nil || page < 1 {
		return Params{}, errors.New("VALIDATION_ERROR", "page must be a positive integer", 422)
	}

	perPage, err := strconv.Atoi(c.Query("per_page", strconv.Itoa(DefaultPerPage)))
	if err != nil || perPage < 1 {
		return Params{}, errors.New("VALIDATION_ERROR", "per_page must be a positive integer", 422)
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return Params{
		Page:    page,
		PerPage: perPage,
		Skip:    (page - 1) * perPage,
		Limit:   perPage,
	}, nil
}

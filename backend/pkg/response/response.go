package response

import (
	"github.com/ecommerce/backend/pkg/errors"
	"github.com/gofiber/fiber/v2"
)

// Response is the standard API response envelope.
type Response struct {
        Success bool        `json:"success"`
        Data    interface{} `json:"data,omitempty"`
        Error   *ErrorBody  `json:"error,omitempty"`
        Meta    *Meta       `json:"meta,omitempty"`
}

type ErrorBody struct {
        Code    string                 `json:"code"`
        Message string                 `json:"message"`
        Details map[string]interface{} `json:"details,omitempty"`
}

type Meta struct {
        Page       int    `json:"page,omitempty"`
        PerPage    int    `json:"per_page,omitempty"`
        Total      int64  `json:"total,omitempty"`
        TotalPages int    `json:"total_pages,omitempty"`
        RequestID  string `json:"request_id,omitempty"`
}

// --- Success responses ---

func OK(c *fiber.Ctx, data interface{}) error {
        meta := requestIDMeta(c)
        return c.JSON(Response{Success: true, Data: data, Meta: meta})
}

func OKWithMeta(c *fiber.Ctx, data interface{}, meta *Meta) error {
        if rid := c.Locals("request_id"); rid != nil {
                if meta == nil {
                        meta = &Meta{}
                }
                meta.RequestID = rid.(string)
        }
        return c.JSON(Response{Success: true, Data: data, Meta: meta})
}

func Created(c *fiber.Ctx, data interface{}) error {
        meta := requestIDMeta(c)
        return c.Status(fiber.StatusCreated).JSON(Response{Success: true, Data: data, Meta: meta})
}

func NoContent(c *fiber.Ctx) error {
        return c.SendStatus(fiber.StatusNoContent)
}

// requestIDMeta extracts the request_id from Fiber locals into a Meta.
// Returns nil if no request_id is set (shouldn't happen with RequestID middleware).
func requestIDMeta(c *fiber.Ctx) *Meta {
        if rid := c.Locals("request_id"); rid != nil {
                return &Meta{RequestID: rid.(string)}
        }
        return nil
}

// --- Error responses ---

func Error(c *fiber.Ctx, err error) error {
        appErr, ok := errors.IsAppError(err)
        if !ok {
                appErr = errors.FromError(err)
        }

        resp := Response{
                Success: false,
                Error: &ErrorBody{
                        Code:    appErr.Code,
                        Message: appErr.Message,
                        Details: appErr.Meta,
                },
        }

        return c.Status(appErr.Status).JSON(resp)
}

// --- Pagination helper ---

func PaginatedMeta(page, perPage int, total int64) *Meta {
        totalPages := int(total) / perPage
        if int(total)%perPage > 0 {
                totalPages++
        }
        return &Meta{
                Page:       page,
                PerPage:    perPage,
                Total:      total,
                TotalPages: totalPages,
        }
}

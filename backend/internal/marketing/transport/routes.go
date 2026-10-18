package transport

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterMarketingRoutes mounts marketing admin endpoints on the given router.
//
// Admin: POST   /api/v1/admin/marketing/campaigns
// Admin: GET    /api/v1/admin/marketing/campaigns
// Admin: GET    /api/v1/admin/marketing/campaigns/:id
// Admin: PUT    /api/v1/admin/marketing/campaigns/:id
// Admin: PATCH  /api/v1/admin/marketing/campaigns/:id/status
// Admin: POST   /api/v1/admin/marketing/segments
// Admin: GET    /api/v1/admin/marketing/segments
// Admin: GET    /api/v1/admin/marketing/segments/:id
// Admin: PUT    /api/v1/admin/marketing/segments/:id
// Admin: DELETE /api/v1/admin/marketing/segments/:id
// Admin: POST   /api/v1/admin/marketing/promotions
// Admin: GET    /api/v1/admin/marketing/promotions
// Admin: GET    /api/v1/admin/marketing/promotions/:id
// Admin: GET    /api/v1/admin/marketing/promotions/code/:code
// Admin: PUT    /api/v1/admin/marketing/promotions/:id
// Admin: PATCH  /api/v1/admin/marketing/promotions/:id/status
func RegisterMarketingRoutes(router fiber.Router, handler *Handler, authMW fiber.Handler, adminMW fiber.Handler) {
	mkt := router.Group("/api/v1/admin/marketing", authMW, adminMW)

	// Campaigns
	mkt.Post("/campaigns", handler.CreateCampaign)
	mkt.Get("/campaigns", handler.ListCampaigns)
	mkt.Get("/campaigns/:id", handler.GetCampaign)
	mkt.Put("/campaigns/:id", handler.UpdateCampaign)
	mkt.Patch("/campaigns/:id/status", handler.TransitionCampaign)

	// Segments
	mkt.Post("/segments", handler.CreateSegment)
	mkt.Get("/segments", handler.ListSegments)
	mkt.Get("/segments/:id", handler.GetSegment)
	mkt.Put("/segments/:id", handler.UpdateSegment)
	mkt.Delete("/segments/:id", handler.DeleteSegment)

	// Promotions
	mkt.Post("/promotions", handler.CreatePromotion)
	mkt.Get("/promotions", handler.ListPromotions)
	mkt.Get("/promotions/:id", handler.GetPromotion)
	mkt.Get("/promotions/code/:code", handler.GetPromotionByCode)
	mkt.Put("/promotions/:id", handler.UpdatePromotion)
	mkt.Patch("/promotions/:id/status", handler.TransitionPromotion)
}

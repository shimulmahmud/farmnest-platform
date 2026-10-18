package transport

import (
	"time"

	"github.com/ecommerce/backend/internal/shipping/domain"
)

// --- Request DTOs ---

// CalculateRateRequest is the JSON body for POST /api/v1/shipping/rates.
type CalculateRateRequest struct {
	Destination *ShipmentAddressRequest  `json:"destination" validate:"required"`
	Packages    []ShipmentPackageRequest `json:"packages" validate:"required"`
	Currency    string                  `json:"currency"`
}

// ShipmentAddressRequest is an address in a request body.
type ShipmentAddressRequest struct {
	Line1      string `json:"line1" validate:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" validate:"required"`
	State      string `json:"state" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required"`
	Country    string `json:"country" validate:"required"`
}

// ShipmentPackageRequest is a package description in a request body.
type ShipmentPackageRequest struct {
	PackageID    string `json:"package_id" validate:"required"`
	WeightGrams int    `json:"weight_grams" validate:"required"`
	LengthCm    int    `json:"length_cm" validate:"required"`
	WidthCm     int    `json:"width_cm" validate:"required"`
	HeightCm    int    `json:"height_cm" validate:"required"`
}

// CreateShipmentRequest is the JSON body for POST /api/v1/admin/shipments.
type CreateShipmentRequest struct {
	OrderID     string                   `json:"order_id" validate:"required"`
	CustomerID  string                   `json:"customer_id" validate:"required"`
	Method      string                   `json:"method" validate:"required"`
	Carrier     string                   `json:"carrier" validate:"required"`
	Destination *ShipmentAddressRequest  `json:"destination" validate:"required"`
	Packages    []ShipmentPackageRequest `json:"packages" validate:"required"`
	Currency    string                   `json:"currency"`
}

// UpdateStatusRequest is the JSON body for PATCH /api/v1/admin/shipments/:id/status.
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required"`
	Reason string `json:"reason"`
}

// CancelShipmentRequest is the JSON body for POST /api/v1/admin/shipments/:id/cancel.
type CancelShipmentRequest struct {
	Reason string `json:"reason"`
}

// --- Response DTOs ---

// ShipmentResponse is the shipment data returned in API responses.
type ShipmentResponse struct {
	ID                string                    `json:"id"`
	ShipmentNumber    string                    `json:"shipment_number"`
	OrderID           string                    `json:"order_id"`
	CustomerID        string                    `json:"customer_id"`
	Method            string                    `json:"method"`
	Carrier           string                    `json:"carrier"`
	TrackingNumber    string                    `json:"tracking_number,omitempty"`
	ProviderRef       string                    `json:"provider_ref,omitempty"`
	Destination       *ShipmentAddressResponse   `json:"destination"`
	Packages          []ShipmentPackageResponse  `json:"packages"`
	CostCents         int                       `json:"cost_cents"`
	Currency          string                    `json:"currency"`
	EstimatedDays     int                       `json:"estimated_days"`
	EstimatedDelivery string                    `json:"estimated_delivery,omitempty"`
	ActualDelivery    string                    `json:"actual_delivery,omitempty"`
	TrackingEvents    []TrackingEventResponse   `json:"tracking_events"`
	Status            string                    `json:"status"`
	StatusHistory     []StatusHistoryResponse   `json:"status_history"`
	CreatedAt         string                    `json:"created_at"`
	UpdatedAt         string                    `json:"updated_at"`
}

// ShippingRateResponse is a rate estimate returned in API responses.
type ShippingRateResponse struct {
	Method        string `json:"method"`
	Carrier       string `json:"carrier"`
	ServiceLevel  string `json:"service_level"`
	CostCents     int    `json:"cost_cents"`
	Currency      string `json:"currency"`
	EstimatedDays int    `json:"estimated_days"`
	EstimatedDate string `json:"estimated_date,omitempty"`
}

// ShipmentAddressResponse is an address in a response.
type ShipmentAddressResponse struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// ShipmentPackageResponse is a package in a response.
type ShipmentPackageResponse struct {
	PackageID    string `json:"package_id"`
	WeightGrams int    `json:"weight_grams"`
	LengthCm    int    `json:"length_cm"`
	WidthCm     int    `json:"width_cm"`
	HeightCm    int    `json:"height_cm"`
}

// TrackingEventResponse is a tracking update in a response.
type TrackingEventResponse struct {
	Status      string `json:"status"`
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
	Timestamp   string `json:"timestamp"`
}

// StatusHistoryResponse is a status change entry in a response.
type StatusHistoryResponse struct {
	Status    string `json:"status"`
	ChangedAt string `json:"changed_at"`
	Reason    string `json:"reason"`
}

// --- Mappers ---

func toShipmentResponse(s *domain.Shipment) ShipmentResponse {
	history := make([]StatusHistoryResponse, len(s.StatusHistory))
	for i, h := range s.StatusHistory {
		history[i] = StatusHistoryResponse{
			Status:    string(h.Status),
			ChangedAt: h.ChangedAt.UTC().Format(time.RFC3339),
			Reason:    h.Reason,
		}
	}

	events := make([]TrackingEventResponse, len(s.TrackingEvents))
	for i, e := range s.TrackingEvents {
		events[i] = TrackingEventResponse{
			Status:      e.Status,
			Description: e.Description,
			Location:    e.Location,
			Timestamp:   e.Timestamp.UTC().Format(time.RFC3339),
		}
	}

	pkgs := make([]ShipmentPackageResponse, len(s.Packages))
	for i, p := range s.Packages {
		pkgs[i] = ShipmentPackageResponse{
			PackageID:    p.PackageID,
			WeightGrams: p.WeightGrams,
			LengthCm:    p.LengthCm,
			WidthCm:     p.WidthCm,
			HeightCm:    p.HeightCm,
		}
	}

	resp := ShipmentResponse{
		ID:             s.ID,
		ShipmentNumber: s.ShipmentNumber,
		OrderID:        s.OrderID,
		CustomerID:     s.CustomerID,
		Method:         string(s.Method),
		Carrier:        s.Carrier,
		TrackingNumber: s.TrackingNumber,
		ProviderRef:    s.ProviderRef,
		Destination:    toShipmentAddressResponse(s.Destination),
		Packages:       pkgs,
		CostCents:      s.CostCents,
		Currency:       s.Currency,
		EstimatedDays:  s.EstimatedDays,
		TrackingEvents: events,
		Status:         string(s.Status),
		StatusHistory:  history,
		CreatedAt:      s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      s.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if s.EstimatedDelivery != nil {
		resp.EstimatedDelivery = s.EstimatedDelivery.UTC().Format(time.RFC3339)
	}
	if s.ActualDelivery != nil {
		resp.ActualDelivery = s.ActualDelivery.UTC().Format(time.RFC3339)
	}

	return resp
}

func toShipmentAddressResponse(a *domain.ShipmentAddress) *ShipmentAddressResponse {
	if a == nil {
		return nil
	}
	return &ShipmentAddressResponse{
		Line1:      a.Line1,
		Line2:      a.Line2,
		City:       a.City,
		State:      a.State,
		PostalCode: a.PostalCode,
		Country:    a.Country,
	}
}

func toDomainShipmentAddress(a *ShipmentAddressRequest) *domain.ShipmentAddress {
	if a == nil {
		return nil
	}
	return &domain.ShipmentAddress{
		Line1:      a.Line1,
		Line2:      a.Line2,
		City:       a.City,
		State:      a.State,
		PostalCode: a.PostalCode,
		Country:    a.Country,
	}
}

func toDomainShipmentPackages(pkgs []ShipmentPackageRequest) []domain.ShipmentPackage {
	result := make([]domain.ShipmentPackage, len(pkgs))
	for i, p := range pkgs {
		result[i] = domain.ShipmentPackage{
			PackageID:    p.PackageID,
			WeightGrams: p.WeightGrams,
			LengthCm:    p.LengthCm,
			WidthCm:     p.WidthCm,
			HeightCm:    p.HeightCm,
		}
	}
	return result
}

func toRateResponse(r *domain.ShippingRate) ShippingRateResponse {
	resp := ShippingRateResponse{
		Method:        string(r.Method),
		Carrier:       r.Carrier,
		ServiceLevel:  r.ServiceLevel,
		CostCents:     r.CostCents,
		Currency:      r.Currency,
		EstimatedDays: r.EstimatedDays,
	}
	if r.EstimatedDate != nil {
		resp.EstimatedDate = r.EstimatedDate.UTC().Format(time.RFC3339)
	}
	return resp
}

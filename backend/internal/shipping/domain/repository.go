package domain

import (
	"context"

	"github.com/ecommerce/backend/pkg/pagination"
)

// ShipmentRepository defines the persistence interface for shipments.
type ShipmentRepository interface {
	Create(ctx context.Context, shipment *Shipment) error
	FindByID(ctx context.Context, id string) (*Shipment, error)
	FindByOrderID(ctx context.Context, orderID string) (*Shipment, error)
	List(ctx context.Context, filter *ShipmentFilter, p pagination.Params) ([]*Shipment, int64, error)
	UpdateStatus(ctx context.Context, id string, fromStatus, toStatus ShipmentStatus, reason string) (*Shipment, error)
	UpdateTracking(ctx context.Context, id string, trackingNumber string, events []TrackingEvent) error
}

// ShippingProvider defines the external provider abstraction for shipping operations.
type ShippingProvider interface {
	// CalculateRate returns available shipping rates for the given parameters.
	CalculateRate(ctx context.Context, req *RateRequest) ([]*ShippingRate, error)
	// CreateShipment creates a shipment with the external provider and returns a provider reference.
	CreateShipment(ctx context.Context, req *ProviderShipmentRequest) (*ProviderShipmentResult, error)
	// TrackShipment retrieves tracking information from the provider.
	TrackShipment(ctx context.Context, providerRef string) ([]TrackingEvent, error)
	// CancelShipment cancels a shipment with the external provider.
	CancelShipment(ctx context.Context, providerRef string) error
}

// EventPublisher defines the interface for publishing shipment events.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, event *ShippingEvent) error
}

// RateRequest contains the parameters for calculating shipping rates.
type RateRequest struct {
	Destination *ShipmentAddress `json:"destination"`
	Packages    []ShipmentPackage `json:"packages"`
	Currency    string           `json:"currency"`
}

// ProviderShipmentRequest contains the parameters for creating a shipment with the provider.
type ProviderShipmentRequest struct {
	ShipmentID   string            `json:"shipment_id"`
	OrderID      string            `json:"order_id"`
	Method       ShippingMethod     `json:"method"`
	Carrier      string            `json:"carrier"`
	Destination  *ShipmentAddress  `json:"destination"`
	Packages     []ShipmentPackage `json:"packages"`
	Currency     string            `json:"currency"`
}

// ProviderShipmentResult contains the response from creating a shipment with the provider.
type ProviderShipmentResult struct {
	ProviderRef    string `json:"provider_ref"`
	TrackingNumber string `json:"tracking_number"`
	LabelURL       string `json:"label_url,omitempty"`
	CostCents      int    `json:"cost_cents"`
}

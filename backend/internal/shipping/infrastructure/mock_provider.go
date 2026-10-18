package infrastructure

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ecommerce/backend/internal/shipping/domain"
)

// LogShippingProvider is a development provider that logs to stdout.
// It returns sensible defaults without calling any external service.
type LogShippingProvider struct {
	carrier string
}

// NewLogShippingProvider creates a new log-based shipping provider.
func NewLogShippingProvider(carrier string) *LogShippingProvider {
	return &LogShippingProvider{carrier: carrier}
}

// CalculateRate returns fixed rates for each shipping method.
func (p *LogShippingProvider) CalculateRate(_ context.Context, req *domain.RateRequest) ([]*domain.ShippingRate, error) {
	fmt.Printf("[shipping] calculate rate: carrier=%s destination=%s packages=%d currency=%s\n",
		p.carrier, req.Destination.PostalCode, len(req.Packages), req.Currency)

	now := time.Now().UTC()
	return []*domain.ShippingRate{
		{
			Method:        domain.ShippingMethodStandard,
			Carrier:       p.carrier,
			ServiceLevel:  "Standard",
			CostCents:     599,
			Currency:      req.Currency,
			EstimatedDays: 7,
			EstimatedDate: ptrTime(now.AddDate(0, 0, 7)),
		},
		{
			Method:        domain.ShippingMethodExpress,
			Carrier:       p.carrier,
			ServiceLevel:  "Express",
			CostCents:     1299,
			Currency:      req.Currency,
			EstimatedDays: 3,
			EstimatedDate: ptrTime(now.AddDate(0, 0, 3)),
		},
		{
			Method:        domain.ShippingMethodOvernight,
			Carrier:       p.carrier,
			ServiceLevel:  "Overnight",
			CostCents:     2499,
			Currency:      req.Currency,
			EstimatedDays: 1,
			EstimatedDate: ptrTime(now.AddDate(0, 0, 1)),
		},
	}, nil
}

// CreateShipment returns a fake provider reference and tracking number.
func (p *LogShippingProvider) CreateShipment(_ context.Context, req *domain.ProviderShipmentRequest) (*domain.ProviderShipmentResult, error) {
	fmt.Printf("[shipping] create shipment: carrier=%s method=%s shipment_id=%s\n",
		p.carrier, req.Method, req.ShipmentID)

	return &domain.ProviderShipmentResult{
		ProviderRef:    fmt.Sprintf("PROV-%s", req.ShipmentID[:8]),
		TrackingNumber: fmt.Sprintf("TRK-%d", time.Now().UnixNano()),
		CostCents:      costForMethod(req.Method),
	}, nil
}

// TrackShipment returns a simulated tracking timeline.
func (p *LogShippingProvider) TrackShipment(_ context.Context, providerRef string) ([]domain.TrackingEvent, error) {
	fmt.Printf("[shipping] track shipment: carrier=%s provider_ref=%s\n", p.carrier, providerRef)

	now := time.Now().UTC()
	return []domain.TrackingEvent{
		{
			Status:      "LABEL_CREATED",
			Description: "Shipping label created",
			Location:    "Warehouse",
			Timestamp:   now.Add(-2 * time.Hour),
		},
		{
			Status:      "PICKED_UP",
			Description: "Package picked up by carrier",
			Location:    "Local hub",
			Timestamp:   now.Add(-1 * time.Hour),
		},
	}, nil
}

// CancelShipment logs the cancellation.
func (p *LogShippingProvider) CancelShipment(_ context.Context, providerRef string) error {
	fmt.Printf("[shipping] cancel shipment: carrier=%s provider_ref=%s\n", p.carrier, providerRef)
	return nil
}

// MockShippingProvider is a test double that records calls and supports injected errors.
type MockShippingProvider struct {
	CalculateRateFn func(ctx context.Context, req *domain.RateRequest) ([]*domain.ShippingRate, error)
	CreateShipmentFn func(ctx context.Context, req *domain.ProviderShipmentRequest) (*domain.ProviderShipmentResult, error)
	TrackShipmentFn  func(ctx context.Context, providerRef string) ([]domain.TrackingEvent, error)
	CancelShipmentFn func(ctx context.Context, providerRef string) error

	CalculateRateCalled atomic.Int64
	CreateShipmentCalled atomic.Int64
	TrackShipmentCalled  atomic.Int64
	CancelShipmentCalled atomic.Int64

	LastRateReq    *domain.RateRequest
	LastShipReq    *domain.ProviderShipmentRequest
	LastTrackRef   string
	LastCancelRef  string
}

// NewMockShippingProvider creates a new mock shipping provider with sensible defaults.
func NewMockShippingProvider() *MockShippingProvider {
	p := &MockShippingProvider{}
	p.CalculateRateFn = p.defaultCalculateRate
	p.CreateShipmentFn = p.defaultCreateShipment
	p.TrackShipmentFn = p.defaultTrackShipment
	p.CancelShipmentFn = p.defaultCancelShipment
	return p
}

func (p *MockShippingProvider) CalculateRate(ctx context.Context, req *domain.RateRequest) ([]*domain.ShippingRate, error) {
	p.CalculateRateCalled.Add(1)
	p.LastRateReq = req
	return p.CalculateRateFn(ctx, req)
}

func (p *MockShippingProvider) CreateShipment(ctx context.Context, req *domain.ProviderShipmentRequest) (*domain.ProviderShipmentResult, error) {
	p.CreateShipmentCalled.Add(1)
	p.LastShipReq = req
	return p.CreateShipmentFn(ctx, req)
}

func (p *MockShippingProvider) TrackShipment(ctx context.Context, providerRef string) ([]domain.TrackingEvent, error) {
	p.TrackShipmentCalled.Add(1)
	p.LastTrackRef = providerRef
	return p.TrackShipmentFn(ctx, providerRef)
}

func (p *MockShippingProvider) CancelShipment(ctx context.Context, providerRef string) error {
	p.CancelShipmentCalled.Add(1)
	p.LastCancelRef = providerRef
	return p.CancelShipmentFn(ctx, providerRef)
}

func (p *MockShippingProvider) defaultCalculateRate(_ context.Context, req *domain.RateRequest) ([]*domain.ShippingRate, error) {
	now := time.Now().UTC()
	return []*domain.ShippingRate{
		{Method: domain.ShippingMethodStandard, Carrier: "mock-carrier", ServiceLevel: "Standard", CostCents: 599, Currency: req.Currency, EstimatedDays: 7, EstimatedDate: ptrTime(now.AddDate(0, 0, 7))},
		{Method: domain.ShippingMethodExpress, Carrier: "mock-carrier", ServiceLevel: "Express", CostCents: 1299, Currency: req.Currency, EstimatedDays: 3, EstimatedDate: ptrTime(now.AddDate(0, 0, 3))},
	}, nil
}

func (p *MockShippingProvider) defaultCreateShipment(_ context.Context, req *domain.ProviderShipmentRequest) (*domain.ProviderShipmentResult, error) {
	return &domain.ProviderShipmentResult{
		ProviderRef:    fmt.Sprintf("PROV-%s", req.ShipmentID[:8]),
		TrackingNumber: fmt.Sprintf("TRK-%d", time.Now().UnixNano()),
		CostCents:      costForMethod(req.Method),
	}, nil
}

func (p *MockShippingProvider) defaultTrackShipment(_ context.Context, _ string) ([]domain.TrackingEvent, error) {
	now := time.Now().UTC()
	return []domain.TrackingEvent{
		{Status: "LABEL_CREATED", Description: "Shipping label created", Location: "Warehouse", Timestamp: now.Add(-2 * time.Hour)},
	}, nil
}

func (p *MockShippingProvider) defaultCancelShipment(_ context.Context, _ string) error {
	return nil
}

func costForMethod(m domain.ShippingMethod) int {
	switch m {
	case domain.ShippingMethodStandard:
		return 599
	case domain.ShippingMethodExpress:
		return 1299
	case domain.ShippingMethodOvernight:
		return 2499
	case domain.ShippingMethodSameDay:
		return 3499
	default:
		return 599
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

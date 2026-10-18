package application

import (
        "context"

        "github.com/ecommerce/backend/internal/shipping/domain"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
)

// Service implements shipping use cases.
type Service struct {
        shipments domain.ShipmentRepository
        provider  domain.ShippingProvider
        events    domain.EventPublisher
}

// NewService creates a shipping application service.
func NewService(shipments domain.ShipmentRepository, provider domain.ShippingProvider, events domain.EventPublisher) *Service {
        return &Service{
                shipments: shipments,
                provider:  provider,
                events:    events,
        }
}

// --- Input structs ---

// CreateShipmentInput holds the data needed to create a new shipment.
type CreateShipmentInput struct {
        OrderID     string               `json:"order_id" validate:"required"`
        CustomerID  string               `json:"customer_id" validate:"required"`
        Method      domain.ShippingMethod `json:"method" validate:"required"`
        Carrier     string               `json:"carrier" validate:"required"`
        Destination *domain.ShipmentAddress `json:"destination" validate:"required"`
        Packages    []domain.ShipmentPackage `json:"packages" validate:"required"`
        Currency    string               `json:"currency"`
}

// RateRequestInput holds the data for calculating shipping rates.
type RateRequestInput struct {
        Destination *domain.ShipmentAddress  `json:"destination" validate:"required"`
        Packages    []domain.ShipmentPackage `json:"packages" validate:"required"`
        Currency    string                  `json:"currency"`
}

// UpdateStatusInput holds the data for an admin status transition.
type UpdateStatusInput struct {
        Status string `json:"status" validate:"required"`
        Reason string `json:"reason"`
}

// --- Use cases ---

// CalculateRate returns available shipping rates from the provider.
func (s *Service) CalculateRate(ctx context.Context, input RateRequestInput) ([]*domain.ShippingRate, error) {
        if input.Destination == nil {
                return nil, errors.New("VALIDATION_ERROR", "destination is required", 422)
        }
        if len(input.Packages) == 0 {
                return nil, errors.New("VALIDATION_ERROR", "at least one package is required", 422)
        }

        currency := input.Currency
        if currency == "" {
                currency = "BDT"
        }

        rates, err := s.provider.CalculateRate(ctx, &domain.RateRequest{
                Destination: input.Destination,
                Packages:    input.Packages,
                Currency:    currency,
        })
        if err != nil {
                return nil, errors.Wrap("PROVIDER_ERROR", "failed to calculate shipping rates", 502, err)
        }
        return rates, nil
}

// CreateShipment creates a new shipment and submits it to the provider,
// transitioning PENDING -> PROCESSING and publishing shipment.created.
func (s *Service) CreateShipment(ctx context.Context, input CreateShipmentInput) (*domain.Shipment, error) {
        if input.OrderID == "" {
                return nil, errors.New("VALIDATION_ERROR", "order_id is required", 422)
        }
        if input.CustomerID == "" {
                return nil, errors.New("VALIDATION_ERROR", "customer_id is required", 422)
        }
        if !domain.ValidShippingMethods[input.Method] {
                return nil, errors.New("VALIDATION_ERROR", "invalid shipping method", 422)
        }
        if input.Carrier == "" {
                return nil, errors.New("VALIDATION_ERROR", "carrier is required", 422)
        }
        if input.Destination == nil {
                return nil, errors.New("VALIDATION_ERROR", "destination is required", 422)
        }
        if len(input.Packages) == 0 {
                return nil, errors.New("VALIDATION_ERROR", "at least one package is required", 422)
        }

        currency := input.Currency
        if currency == "" {
                currency = "BDT"
        }

        // Create the shipment entity in PENDING status
        shipment := domain.NewShipment(
                input.OrderID,
                input.CustomerID,
                input.Method,
                input.Carrier,
                input.Destination,
                input.Packages,
                0, // cost set by provider
                currency,
                estimatedDaysForMethod(input.Method),
        )

        // Submit to provider
        providerResult, err := s.provider.CreateShipment(ctx, &domain.ProviderShipmentRequest{
                ShipmentID:  shipment.ID,
                OrderID:     input.OrderID,
                Method:      input.Method,
                Carrier:     input.Carrier,
                Destination: input.Destination,
                Packages:    input.Packages,
                Currency:    currency,
        })
        if err != nil {
                shipment.ProviderError = err.Error()
                _ = s.shipments.Create(ctx, shipment)
                return nil, errors.Wrap("PROVIDER_ERROR", "failed to create shipment with provider", 502, err)
        }

        // Update with provider results
        shipment.ProviderRef = providerResult.ProviderRef
        shipment.TrackingNumber = providerResult.TrackingNumber
        shipment.CostCents = providerResult.CostCents

        // Persist in PENDING status first
        if err := s.shipments.Create(ctx, shipment); err != nil {
                return nil, err
        }

        // Transition PENDING -> PROCESSING
        fromStatus := shipment.Status
        updated, err := s.shipments.UpdateStatus(ctx, shipment.ID, domain.ShipmentPending, domain.ShipmentProcessing, "submitted to carrier")
        if err != nil {
                return nil, errors.Wrap("INTERNAL_ERROR", "shipment created but failed to transition to processing", 500, err)
        }

        // Publish shipment.created event
        s.publishEvent(ctx, "shipment.created", &domain.ShippingEvent{
                EventType:      "shipment.created",
                ShipmentID:     updated.ID,
                ShipmentNumber: updated.ShipmentNumber,
                OrderID:        updated.OrderID,
                CustomerID:     updated.CustomerID,
                FromStatus:     string(fromStatus),
                ToStatus:       string(domain.ShipmentProcessing),
                TrackingNumber: updated.TrackingNumber,
                Carrier:        updated.Carrier,
                ProviderRef:    updated.ProviderRef,
        })

        return updated, nil
}

// GetShipment retrieves a shipment by ID.
func (s *Service) GetShipment(ctx context.Context, id string) (*domain.Shipment, error) {
        return s.shipments.FindByID(ctx, id)
}

// GetShipmentByOrder retrieves a shipment by order ID.
func (s *Service) GetShipmentByOrder(ctx context.Context, orderID string) (*domain.Shipment, error) {
        if orderID == "" {
                return nil, errors.New("VALIDATION_ERROR", "order_id is required", 422)
        }
        return s.shipments.FindByOrderID(ctx, orderID)
}

// ListShipments returns paginated shipments for a specific customer.
func (s *Service) ListShipments(ctx context.Context, customerID, status string, p pagination.Params) ([]*domain.Shipment, int64, error) {
        if customerID == "" {
                return nil, 0, errors.New("VALIDATION_ERROR", "customer_id is required", 422)
        }
        filter := &domain.ShipmentFilter{
                CustomerID: customerID,
                Status:     status,
        }
        return s.shipments.List(ctx, filter, p)
}

// ListAllShipments returns paginated shipments for admin.
func (s *Service) ListAllShipments(ctx context.Context, status, customerID, carrier string, p pagination.Params) ([]*domain.Shipment, int64, error) {
        filter := &domain.ShipmentFilter{
                CustomerID: customerID,
                Status:     status,
                Carrier:    carrier,
        }
        return s.shipments.List(ctx, filter, p)
}

// TrackShipment retrieves tracking information for a shipment.
func (s *Service) TrackShipment(ctx context.Context, id string) (*domain.Shipment, error) {
        shipment, err := s.shipments.FindByID(ctx, id)
        if err != nil {
                return nil, err
        }

        if shipment.ProviderRef == "" {
                // No provider ref yet, return as-is
                return shipment, nil
        }

        // Fetch tracking from provider
        events, err := s.provider.TrackShipment(ctx, shipment.ProviderRef)
        if err != nil {
                return nil, errors.Wrap("PROVIDER_ERROR", "failed to track shipment with provider", 502, err)
        }

        // Persist tracking events
        if err := s.shipments.UpdateTracking(ctx, id, shipment.TrackingNumber, events); err != nil {
                return nil, err
        }

        // Reload to get updated tracking
        shipment, err = s.shipments.FindByID(ctx, id)
        if err != nil {
                return nil, err
        }

        // If the provider returned a delivered event, auto-transition
        for _, evt := range events {
                if evt.Status == "DELIVERED" && shipment.CanTransitionTo(domain.ShipmentDelivered) {
                        fromStatus := shipment.Status
                        updated, err := s.shipments.UpdateStatus(ctx, id, shipment.Status, domain.ShipmentDelivered, "delivered per provider")
                        if err == nil {
                                s.publishEvent(ctx, "shipment.delivered", &domain.ShippingEvent{
                                        EventType:      "shipment.delivered",
                                        ShipmentID:     updated.ID,
                                        ShipmentNumber: updated.ShipmentNumber,
                                        OrderID:        updated.OrderID,
                                        CustomerID:     updated.CustomerID,
                                        FromStatus:     string(fromStatus),
                                        ToStatus:       string(domain.ShipmentDelivered),
                                        TrackingNumber: updated.TrackingNumber,
                                        Carrier:        updated.Carrier,
                                        ProviderRef:    updated.ProviderRef,
                                })
                                return updated, nil
                        }
                }
                if evt.Status == "IN_TRANSIT" && shipment.CanTransitionTo(domain.ShipmentShipped) {
                        fromStatus := shipment.Status
                        updated, err := s.shipments.UpdateStatus(ctx, id, shipment.Status, domain.ShipmentShipped, "in transit per provider")
                        if err == nil {
                                s.publishEvent(ctx, "shipment.shipped", &domain.ShippingEvent{
                                        EventType:      "shipment.shipped",
                                        ShipmentID:     updated.ID,
                                        ShipmentNumber: updated.ShipmentNumber,
                                        OrderID:        updated.OrderID,
                                        CustomerID:     updated.CustomerID,
                                        FromStatus:     string(fromStatus),
                                        ToStatus:       string(domain.ShipmentShipped),
                                        TrackingNumber: updated.TrackingNumber,
                                        Carrier:        updated.Carrier,
                                        ProviderRef:    updated.ProviderRef,
                                })
                                return updated, nil
                        }
                }
        }

        return shipment, nil
}

// UpdateShipmentStatus performs a status transition with validation.
func (s *Service) UpdateShipmentStatus(ctx context.Context, shipmentID string, input UpdateStatusInput) (*domain.Shipment, error) {
        newStatus := domain.ShipmentStatus(input.Status)

        current, err := s.shipments.FindByID(ctx, shipmentID)
        if err != nil {
                return nil, err
        }

        if !current.CanTransitionTo(newStatus) {
                return nil, errors.New("BAD_REQUEST",
                        "invalid status transition: "+string(current.Status)+" -> "+string(newStatus), 400)
        }

        reason := input.Reason
        if reason == "" {
                reason = "status updated by admin"
        }

        updated, err := s.shipments.UpdateStatus(ctx, shipmentID, current.Status, newStatus, reason)
        if err != nil {
                return nil, err
        }

        eventSubject := domain.TransitionEvent(current.Status, newStatus)
        if eventSubject != "" {
                s.publishEvent(ctx, eventSubject, &domain.ShippingEvent{
                        EventType:      eventSubject,
                        ShipmentID:     updated.ID,
                        ShipmentNumber: updated.ShipmentNumber,
                        OrderID:        updated.OrderID,
                        CustomerID:     updated.CustomerID,
                        FromStatus:     string(current.Status),
                        ToStatus:       string(newStatus),
                        TrackingNumber: updated.TrackingNumber,
                        Carrier:        updated.Carrier,
                        ProviderRef:    updated.ProviderRef,
                })
        }

        return updated, nil
}

// CancelShipment cancels a shipment with both the provider and internally.
func (s *Service) CancelShipment(ctx context.Context, id string, reason string) (*domain.Shipment, error) {
        if reason == "" {
                reason = "cancelled by user"
        }

        current, err := s.shipments.FindByID(ctx, id)
        if err != nil {
                return nil, err
        }

        if !current.CanTransitionTo(domain.ShipmentCancelled) {
                return nil, errors.New("BAD_REQUEST",
                        "cannot cancel shipment in status "+string(current.Status), 400)
        }

        // Cancel with provider if we have a provider ref
        if current.ProviderRef != "" {
                if err := s.provider.CancelShipment(ctx, current.ProviderRef); err != nil {
                        return nil, errors.Wrap("PROVIDER_ERROR", "failed to cancel shipment with provider", 502, err)
                }
        }

        updated, err := s.shipments.UpdateStatus(ctx, id, current.Status, domain.ShipmentCancelled, reason)
        if err != nil {
                return nil, err
        }

        return updated, nil
}

// --- Helpers ---

func (s *Service) publishEvent(ctx context.Context, subject string, event *domain.ShippingEvent) {
        if s.events != nil {
                _ = s.events.Publish(ctx, subject, event)
        }
}

func estimatedDaysForMethod(m domain.ShippingMethod) int {
        switch m {
        case domain.ShippingMethodStandard:
                return 7
        case domain.ShippingMethodExpress:
                return 3
        case domain.ShippingMethodOvernight:
                return 1
        case domain.ShippingMethodSameDay:
                return 0
        default:
                return 7
        }
}

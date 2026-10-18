package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ShipmentStatus represents the lifecycle state of a shipment.
type ShipmentStatus string

const (
	ShipmentPending    ShipmentStatus = "PENDING"
	ShipmentProcessing ShipmentStatus = "PROCESSING"
	ShipmentShipped    ShipmentStatus = "SHIPPED"
	ShipmentDelivered  ShipmentStatus = "DELIVERED"
	ShipmentCancelled  ShipmentStatus = "CANCELLED"
)

// validTransitions defines the only allowed status transitions.
var validTransitions = map[ShipmentStatus][]ShipmentStatus{
	ShipmentPending:    {ShipmentProcessing, ShipmentCancelled},
	ShipmentProcessing: {ShipmentShipped, ShipmentCancelled},
	ShipmentShipped:    {ShipmentDelivered},
	ShipmentDelivered:  {},
	ShipmentCancelled:  {},
}

// transitionEvents maps (from, to) status pairs to NATS event subjects.
var transitionEvents = map[[2]ShipmentStatus]string{
	{ShipmentPending, ShipmentProcessing}: "shipment.created",
	{ShipmentProcessing, ShipmentShipped}: "shipment.shipped",
	{ShipmentShipped, ShipmentDelivered}:  "shipment.delivered",
}

// IsTerminal returns true if the shipment is in a final state.
func (s ShipmentStatus) IsTerminal() bool {
	return s == ShipmentDelivered || s == ShipmentCancelled
}

// CanTransitionTo checks whether transitioning to the target status is allowed.
func (s ShipmentStatus) CanTransitionTo(target ShipmentStatus) bool {
	targets, ok := validTransitions[s]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// TransitionEvent returns the NATS event subject for this transition, or empty if none.
func TransitionEvent(from, to ShipmentStatus) string {
	return transitionEvents[[2]ShipmentStatus{from, to}]
}

// ShippingMethod represents a shipping method type.
type ShippingMethod string

const (
	ShippingMethodStandard ShippingMethod = "STANDARD"
	ShippingMethodExpress  ShippingMethod = "EXPRESS"
	ShippingMethodOvernight ShippingMethod = "OVERNIGHT"
	ShippingMethodSameDay   ShippingMethod = "SAME_DAY"
)

// ValidShippingMethods is the set of allowed shipping method values.
var ValidShippingMethods = map[ShippingMethod]bool{
	ShippingMethodStandard: true,
	ShippingMethodExpress:  true,
	ShippingMethodOvernight: true,
	ShippingMethodSameDay:   true,
}

// ShippingRate holds the result of a rate calculation.
type ShippingRate struct {
	Method          ShippingMethod `json:"method"`
	Carrier         string        `json:"carrier"`
	ServiceLevel    string        `json:"service_level"`
	CostCents       int           `json:"cost_cents"`
	Currency        string        `json:"currency"`
	EstimatedDays   int           `json:"estimated_days"`
	EstimatedDate   *time.Time    `json:"estimated_date,omitempty"`
}

// TrackingEvent represents a single tracking update.
type TrackingEvent struct {
	Status      string    `bson:"status" json:"status"`
	Description string    `bson:"description" json:"description"`
	Location    string    `bson:"location" json:"location,omitempty"`
	Timestamp   time.Time `bson:"timestamp" json:"timestamp"`
}

// ShipmentAddress holds shipping destination information.
type ShipmentAddress struct {
	Line1      string `bson:"line1" json:"line1"`
	Line2      string `bson:"line2" json:"line2,omitempty"`
	City       string `bson:"city" json:"city"`
	State      string `bson:"state" json:"state"`
	PostalCode string `bson:"postal_code" json:"postal_code"`
	Country    string `bson:"country" json:"country"`
}

// ShipmentPackage describes a single package in a shipment.
type ShipmentPackage struct {
	PackageID string `bson:"package_id" json:"package_id"`
	WeightGrams int   `bson:"weight_grams" json:"weight_grams"`
	LengthCm   int   `bson:"length_cm" json:"length_cm"`
	WidthCm    int   `bson:"width_cm" json:"width_cm"`
	HeightCm   int   `bson:"height_cm" json:"height_cm"`
}

// StatusHistoryEntry records a single status change.
type StatusHistoryEntry struct {
	Status    ShipmentStatus `bson:"status" json:"status"`
	ChangedAt time.Time      `bson:"changed_at" json:"changed_at"`
	Reason    string         `bson:"reason" json:"reason"`
}

// Shipment represents a shipping entity.
type Shipment struct {
	ID               string             `bson:"_id" json:"id"`
	ShipmentNumber   string             `bson:"shipment_number" json:"shipment_number"`
	OrderID          string             `bson:"order_id" json:"order_id"`
	CustomerID       string             `bson:"customer_id" json:"customer_id"`
	Method           ShippingMethod     `bson:"method" json:"method"`
	Carrier          string             `bson:"carrier" json:"carrier"`
	TrackingNumber   string             `bson:"tracking_number" json:"tracking_number,omitempty"`
	ProviderRef      string             `bson:"provider_ref" json:"provider_ref,omitempty"`
	Destination      *ShipmentAddress   `bson:"destination" json:"destination"`
	Packages         []ShipmentPackage  `bson:"packages" json:"packages"`
	CostCents        int                `bson:"cost_cents" json:"cost_cents"`
	Currency         string             `bson:"currency" json:"currency"`
	EstimatedDays    int                `bson:"estimated_days" json:"estimated_days"`
	EstimatedDelivery *time.Time         `bson:"estimated_delivery" json:"estimated_delivery,omitempty"`
	ActualDelivery   *time.Time         `bson:"actual_delivery" json:"actual_delivery,omitempty"`
	TrackingEvents   []TrackingEvent    `bson:"tracking_events" json:"tracking_events"`
	Status           ShipmentStatus     `bson:"status" json:"status"`
	StatusHistory    []StatusHistoryEntry `bson:"status_history" json:"status_history"`
	ProviderError    string             `bson:"provider_error" json:"provider_error,omitempty"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

// CanTransitionTo delegates to the ShipmentStatus method.
func (s *Shipment) CanTransitionTo(target ShipmentStatus) bool {
	return s.Status.CanTransitionTo(target)
}

// TransitionTo validates and applies a status transition, appending to history.
// Returns an error if the transition is invalid.
func (s *Shipment) TransitionTo(target ShipmentStatus, reason string) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("invalid transition: %s -> %s", s.Status, target)
	}
	now := time.Now().UTC()
	s.Status = target
	s.StatusHistory = append(s.StatusHistory, StatusHistoryEntry{
		Status:    target,
		ChangedAt: now,
		Reason:    reason,
	})
	s.UpdatedAt = now
	return nil
}

// NewShipment creates a new shipment in PENDING status.
func NewShipment(orderID, customerID string, method ShippingMethod, carrier string, destination *ShipmentAddress, packages []ShipmentPackage, costCents int, currency string, estimatedDays int) *Shipment {
	now := time.Now().UTC()
	var estimatedDelivery *time.Time
	if estimatedDays > 0 {
		ed := now.AddDate(0, 0, estimatedDays)
		estimatedDelivery = &ed
	}
	return &Shipment{
		ID:                uuid.New().String(),
		ShipmentNumber:    generateShipmentNumber(now),
		OrderID:           orderID,
		CustomerID:        customerID,
		Method:            method,
		Carrier:           carrier,
		Destination:       destination,
		Packages:          packages,
		CostCents:         costCents,
		Currency:          currency,
		EstimatedDays:     estimatedDays,
		EstimatedDelivery: estimatedDelivery,
		Status:            ShipmentPending,
		StatusHistory: []StatusHistoryEntry{
			{
				Status:    ShipmentPending,
				ChangedAt: now,
				Reason:    "shipment created",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// generateShipmentNumber creates a human-readable shipment number: SHP-YYYYMMDD-XXXXXXXX.
func generateShipmentNumber(t time.Time) string {
	date := t.UTC().Format("20060102")
	suffix := uuid.New().String()[:8]
	return fmt.Sprintf("SHP-%s-%s", date, suffix)
}

// ShippingEvent is published to NATS on shipment state changes.
type ShippingEvent struct {
	EventType       string `json:"event_type"`
	ShipmentID      string `json:"shipment_id"`
	ShipmentNumber  string `json:"shipment_number"`
	OrderID         string `json:"order_id"`
	CustomerID      string `json:"customer_id"`
	FromStatus      string `json:"from_status"`
	ToStatus        string `json:"to_status"`
	TrackingNumber  string `json:"tracking_number,omitempty"`
	Carrier         string `json:"carrier,omitempty"`
	ProviderRef     string `json:"provider_ref,omitempty"`
}

// ShipmentFilter holds optional filters for listing shipments.
type ShipmentFilter struct {
	OrderID    string
	CustomerID string
	Status     string
	Carrier    string
	From       *time.Time
	To         *time.Time
}

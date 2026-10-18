package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ecommerce/backend/pkg/errors"
)

// Delivery areas. Courier prices in Bangladesh usually differ inside and outside Dhaka city.
const (
	AreaInsideDhaka  = "inside_dhaka"
	AreaOutsideDhaka = "outside_dhaka"
)

// Area is a delivery area the customer can pick at checkout.
type Area struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Areas lists the delivery areas in display order.
func Areas() []Area {
	return []Area{
		{ID: AreaInsideDhaka, Label: "Inside Dhaka city"},
		{ID: AreaOutsideDhaka, Label: "Outside Dhaka"},
	}
}

// Method is one delivery option (for example standard or express). Fees are whole BDT taka.
type Method struct {
	ID              string `bson:"id" json:"id"`
	Label           string `bson:"label" json:"label"`
	Detail          string `bson:"detail" json:"detail"`
	Enabled         bool   `bson:"enabled" json:"enabled"`
	FeeInsideDhaka  int    `bson:"fee_inside_dhaka" json:"fee_inside_dhaka"`
	FeeOutsideDhaka int    `bson:"fee_outside_dhaka" json:"fee_outside_dhaka"`
}

// Delivery is the shop's delivery-charge setting, edited in the admin panel.
type Delivery struct {
	Methods []Method `bson:"methods" json:"methods"`
	// FreeShippingThreshold: item subtotal at or above which STANDARD delivery is free. 0 = never free.
	FreeShippingThreshold int       `bson:"free_shipping_threshold" json:"free_shipping_threshold"`
	UpdatedAt             time.Time `bson:"updated_at" json:"updated_at"`
	UpdatedBy             string    `bson:"updated_by" json:"updated_by"`
}

// Fixed method IDs (the shop edits their label, text and prices, and can switch them on or off).
const (
	MethodStandard = "standard"
	MethodExpress  = "express"
)

// MaxFee is a sanity limit so a typo like 6000000 is caught.
const MaxFee = 100000

// DefaultDelivery is used until the shop saves its own charges.
func DefaultDelivery() Delivery {
	return Delivery{
		Methods: []Method{
			{ID: MethodStandard, Label: "Standard delivery", Detail: "2-4 working days", Enabled: true, FeeInsideDhaka: 60, FeeOutsideDhaka: 120},
			{ID: MethodExpress, Label: "Express delivery", Detail: "Next working day", Enabled: true, FeeInsideDhaka: 120, FeeOutsideDhaka: 200},
		},
		FreeShippingThreshold: 2000,
	}
}

func invalid(msg string) error { return errors.New("VALIDATION_ERROR", msg, 422) }

// Validate checks the setting before it is saved.
func (d *Delivery) Validate() error {
	if len(d.Methods) == 0 {
		return invalid("at least one delivery method is required")
	}
	if d.FreeShippingThreshold < 0 || d.FreeShippingThreshold > 10_000_000 {
		return invalid("free delivery amount must be between 0 and 10,000,000")
	}
	seen := map[string]bool{}
	enabled := 0
	for i := range d.Methods {
		m := &d.Methods[i]
		m.ID = strings.ToLower(strings.TrimSpace(m.ID))
		m.Label = strings.TrimSpace(m.Label)
		m.Detail = strings.TrimSpace(m.Detail)
		if m.ID != MethodStandard && m.ID != MethodExpress {
			return invalid(fmt.Sprintf("unknown delivery method %q", m.ID))
		}
		if seen[m.ID] {
			return invalid("each delivery method can appear only once")
		}
		seen[m.ID] = true
		if m.Label == "" || utf8.RuneCountInString(m.Label) > 40 {
			return invalid("each delivery method needs a name (up to 40 characters)")
		}
		if utf8.RuneCountInString(m.Detail) > 80 {
			return invalid("delivery description is too long (up to 80 characters)")
		}
		if m.FeeInsideDhaka < 0 || m.FeeInsideDhaka > MaxFee || m.FeeOutsideDhaka < 0 || m.FeeOutsideDhaka > MaxFee {
			return invalid(fmt.Sprintf("delivery charges must be between 0 and %d taka", MaxFee))
		}
		if m.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return invalid("at least one delivery method must be switched on")
	}
	return nil
}

// EnabledMethods returns the methods customers can choose, in order.
func (d Delivery) EnabledMethods() []Method {
	out := make([]Method, 0, len(d.Methods))
	for _, m := range d.Methods {
		if m.Enabled {
			out = append(out, m)
		}
	}
	return out
}

// Fee works out the delivery charge for a method, area and item subtotal.
// An empty method means the first enabled one; an empty area means inside Dhaka.
// Standard delivery is free when the subtotal reaches FreeShippingThreshold.
func (d Delivery) Fee(method, area string, subtotal int) (methodID string, fee int, err error) {
	method = strings.ToLower(strings.TrimSpace(method))
	area = strings.ToLower(strings.TrimSpace(area))
	if area == "" {
		area = AreaInsideDhaka
	}
	if area != AreaInsideDhaka && area != AreaOutsideDhaka {
		return "", 0, invalid("unknown delivery area: " + area)
	}
	enabled := d.EnabledMethods()
	if len(enabled) == 0 {
		return "", 0, invalid("delivery is not available right now")
	}
	var chosen *Method
	if method == "" {
		chosen = &enabled[0]
	} else {
		for i := range enabled {
			if enabled[i].ID == method {
				chosen = &enabled[i]
				break
			}
		}
	}
	if chosen == nil {
		return "", 0, invalid("this delivery method is not available: " + method)
	}
	if chosen.ID == MethodStandard && d.FreeShippingThreshold > 0 && subtotal >= d.FreeShippingThreshold {
		return chosen.ID, 0, nil
	}
	if area == AreaOutsideDhaka {
		return chosen.ID, chosen.FeeOutsideDhaka, nil
	}
	return chosen.ID, chosen.FeeInsideDhaka, nil
}

// Repository stores the delivery setting.
type Repository interface {
	// GetDelivery returns the saved setting, or nil when none was saved yet.
	GetDelivery(ctx context.Context) (*Delivery, error)
	SaveDelivery(ctx context.Context, d *Delivery) error
}

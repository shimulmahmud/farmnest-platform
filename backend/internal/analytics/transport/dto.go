package transport

import (
        "strconv"
        "time"

        domain "github.com/ecommerce/backend/internal/analytics/domain"
        "github.com/ecommerce/backend/internal/analytics/application"
)

const timestampFormat = "2006-01-02T15:04:05Z"

// --- Request DTOs ---

// GetAnalyticsQuery holds query parameters for analytics endpoints.
type GetAnalyticsQuery struct {
        // Start is the start of the time range (RFC3339 or Unix timestamp). Defaults to 30 days ago.
        Start string `query:"start"`
        // End is the end of the time range (RFC3339 or Unix timestamp). Defaults to now.
        End string `query:"end"`
        // StartAt / EndAt are accepted as aliases of start / end. Plain dates (YYYY-MM-DD) are
        // read as Bangladesh time; an end date includes the whole day.
        StartAt string `query:"start_at"`
        EndAt   string `query:"end_at"`
        // Limit is the maximum number of results for list endpoints.
        Limit string `query:"limit"`
}

// ParseTimeRange converts the query string start/end into a domain.TimeRange.
// It supports RFC3339 format and Unix timestamps.
func (q *GetAnalyticsQuery) ParseTimeRange() (domain.TimeRange, error) {
        var tr domain.TimeRange

        if q.Start == "" {
                q.Start = q.StartAt
        }
        if q.End == "" {
                q.End = q.EndAt
        }

        if q.Start != "" {
                t, err := parseTime(q.Start)
                if err != nil {
                        return tr, err
                }
                tr.Start = t
        } else {
                tr.Start = time.Now().UTC().AddDate(0, 0, -30)
        }

        if q.End != "" {
                t, err := parseTime(q.End)
                if err != nil {
                        return tr, err
                }
                if isDateOnly(q.End) {
                        t = t.Add(24*time.Hour - time.Nanosecond) // include the whole end day
                }
                tr.End = t
        } else {
                tr.End = time.Now().UTC()
        }

        return tr, nil
}

// ParseLimit returns the limit as an integer, defaulting to 10.
func (q *GetAnalyticsQuery) ParseLimit() int {
        if q.Limit == "" {
                return 10
        }
        l, err := strconv.Atoi(q.Limit)
        if err != nil || l <= 0 {
                return 10
        }
        if l > 100 {
                return 100
        }
        return l
}

// --- Response DTOs ---

// OverviewResponse is the JSON response for the analytics overview endpoint.
type OverviewResponse struct {
        ProductViews     int64   `json:"product_views"`
        UniqueViewers    int64   `json:"unique_viewers"`
        CartsCreated     int64   `json:"carts_created"`
        CheckoutsStarted int64   `json:"checkouts_started"`
        OrdersCompleted  int64   `json:"orders_completed"`
        ConversionRate   float64 `json:"conversion_rate"`
        RevenueCents     int64   `json:"revenue_cents"`
        StartAt          string  `json:"start_at"`
        EndAt            string  `json:"end_at"`
}

// TopProductResponse represents a single product in the top-products response.
type TopProductResponse struct {
        ProductID string `json:"product_id"`
        Views     int64  `json:"views"`
}

// DailyCountResponse represents a single day's count in a trend response.
type DailyCountResponse struct {
        Date  string `json:"date"`
        Count int64  `json:"count"`
}

// --- Mappers ---

func toOverviewResponse(r *application.OverviewResult) OverviewResponse {
        return OverviewResponse{
                ProductViews:     r.ProductViews,
                UniqueViewers:    r.UniqueViewers,
                CartsCreated:     r.CartsCreated,
                CheckoutsStarted: r.CheckoutsStarted,
                OrdersCompleted:  r.OrdersCompleted,
                ConversionRate:   r.ConversionRate,
                RevenueCents:     r.RevenueCents,
                StartAt:          r.TimeRange.Start.Format(timestampFormat),
                EndAt:            r.TimeRange.End.Format(timestampFormat),
        }
}

func toTopProductResponse(stat *domain.ProductViewStat) TopProductResponse {
        return TopProductResponse{
                ProductID: stat.ProductID,
                Views:     stat.Views,
        }
}

func toDailyCountResponse(dc *domain.DailyCount) DailyCountResponse {
        return DailyCountResponse{
                Date:  dc.Date,
                Count: dc.Count,
        }
}

func toTopProductResponses(stats []*domain.ProductViewStat) []TopProductResponse {
        if len(stats) == 0 {
                return []TopProductResponse{}
        }
        result := make([]TopProductResponse, len(stats))
        for i, s := range stats {
                result[i] = toTopProductResponse(s)
        }
        return result
}

func toDailyCountResponses(counts []*domain.DailyCount) []DailyCountResponse {
        if len(counts) == 0 {
                return []DailyCountResponse{}
        }
        result := make([]DailyCountResponse, len(counts))
        for i, c := range counts {
                result[i] = toDailyCountResponse(c)
        }
        return result
}

// --- Helpers ---

// parseTime tries to parse a string as RFC3339, falling back to Unix timestamp.
func parseTime(s string) (time.Time, error) {
        // Plain date, Bangladesh time (what the admin date pickers send)
        if t, err := time.ParseInLocation("2006-01-02", s, bangladeshTime); err == nil {
                return t.UTC(), nil
        }

        // Try RFC3339
        if t, err := time.Parse(time.RFC3339, s); err == nil {
                return t.UTC(), nil
        }

        // Try Unix timestamp (seconds)
        if sec, err := strconv.ParseInt(s, 10, 64); err == nil {
                return time.Unix(sec, 0).UTC(), nil
        }

        // Try Unix timestamp (milliseconds)
        if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
                return time.UnixMilli(ms).UTC(), nil
        }

        return time.Time{}, &domain.AnalyticsError{
                Code:    "INVALID_TIME_FORMAT",
                Message: "time must be RFC3339 or Unix timestamp",
        }
}

// bangladeshTime is UTC+6; plain dates in queries are read in this zone.
var bangladeshTime = time.FixedZone("BST", 6*60*60)

// isDateOnly reports whether s is a plain YYYY-MM-DD date.
func isDateOnly(s string) bool {
	_, err := time.ParseInLocation("2006-01-02", s, bangladeshTime)
	return err == nil
}

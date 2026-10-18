package application

import (
        "context"
        "sync"
        "sync/atomic"
        "testing"
        "time"

        "github.com/ecommerce/backend/internal/shipping/domain"
        shipInfra "github.com/ecommerce/backend/internal/shipping/infrastructure"
        "github.com/ecommerce/backend/pkg/errors"
        "github.com/ecommerce/backend/pkg/pagination"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Test Helpers ---

func testAddress() *domain.ShipmentAddress {
        return &domain.ShipmentAddress{
                Line1:      "123 Main St", City: "Springfield", State: "IL", PostalCode: "62701", Country: "US",
        }
}

func testPackages() []domain.ShipmentPackage {
        return []domain.ShipmentPackage{
                {PackageID: "pkg-1", WeightGrams: 500, LengthCm: 20, WidthCm: 15, HeightCm: 10},
        }
}

// --- Mock Repository ---

func copyShipment(s *domain.Shipment) *domain.Shipment {
        cp := *s
        cp.Packages = make([]domain.ShipmentPackage, len(s.Packages))
        copy(cp.Packages, s.Packages)
        cp.TrackingEvents = make([]domain.TrackingEvent, len(s.TrackingEvents))
        copy(cp.TrackingEvents, s.TrackingEvents)
        cp.StatusHistory = make([]domain.StatusHistoryEntry, len(s.StatusHistory))
        copy(cp.StatusHistory, s.StatusHistory)
        if s.Destination != nil {
                dest := *s.Destination
                cp.Destination = &dest
        }
        if s.EstimatedDelivery != nil {
                ed := *s.EstimatedDelivery
                cp.EstimatedDelivery = &ed
        }
        if s.ActualDelivery != nil {
                ad := *s.ActualDelivery
                cp.ActualDelivery = &ad
        }
        return &cp
}

type mockShipmentRepo struct {
        mu        sync.Mutex
        shipments map[string]*domain.Shipment
}

func newMockShipmentRepo() *mockShipmentRepo {
        return &mockShipmentRepo{shipments: make(map[string]*domain.Shipment)}
}

func (r *mockShipmentRepo) Create(_ context.Context, shipment *domain.Shipment) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        if _, ok := r.shipments[shipment.ID]; ok {
                return errors.New("CONFLICT", "shipment already exists", 409)
        }
        r.shipments[shipment.ID] = copyShipment(shipment)
        return nil
}

func (r *mockShipmentRepo) FindByID(_ context.Context, id string) (*domain.Shipment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        s, ok := r.shipments[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        return copyShipment(s), nil
}

func (r *mockShipmentRepo) FindByOrderID(_ context.Context, orderID string) (*domain.Shipment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        for _, s := range r.shipments {
                if s.OrderID == orderID {
                        return copyShipment(s), nil
                }
        }
        return nil, errors.ErrNotFound
}

func (r *mockShipmentRepo) List(_ context.Context, filter *domain.ShipmentFilter, p pagination.Params) ([]*domain.Shipment, int64, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        var result []*domain.Shipment
        for _, s := range r.shipments {
                if filter != nil {
                        if filter.CustomerID != "" && s.CustomerID != filter.CustomerID {
                                continue
                        }
                        if filter.Status != "" && string(s.Status) != filter.Status {
                                continue
                        }
                        if filter.Carrier != "" && s.Carrier != filter.Carrier {
                                continue
                        }
                }
                result = append(result, copyShipment(s))
        }
        total := int64(len(result))
        start := p.Skip
        if start > len(result) {
                start = len(result)
        }
        end := start + p.Limit
        if end > len(result) {
                end = len(result)
        }
        return result[start:end], total, nil
}

func (r *mockShipmentRepo) UpdateStatus(_ context.Context, id string, fromStatus, toStatus domain.ShipmentStatus, reason string) (*domain.Shipment, error) {
        r.mu.Lock()
        defer r.mu.Unlock()
        s, ok := r.shipments[id]
        if !ok {
                return nil, errors.ErrNotFound
        }
        if s.Status != fromStatus {
                return nil, errors.New("CONFLICT", "shipment not found or status has changed", 409)
        }
        s.Status = toStatus
        s.StatusHistory = append(s.StatusHistory, domain.StatusHistoryEntry{
                Status: toStatus, ChangedAt: time.Now().UTC(), Reason: reason,
        })
        s.UpdatedAt = time.Now().UTC()
        return copyShipment(s), nil
}

func (r *mockShipmentRepo) UpdateTracking(_ context.Context, id string, trackingNumber string, events []domain.TrackingEvent) error {
        r.mu.Lock()
        defer r.mu.Unlock()
        s, ok := r.shipments[id]
        if !ok {
                return errors.ErrNotFound
        }
        s.TrackingNumber = trackingNumber
        s.TrackingEvents = append(s.TrackingEvents, events...)
        s.UpdatedAt = time.Now().UTC()
        return nil
}

// --- Mock Event Publisher ---

type mockEventPublisher struct {
        mu      sync.Mutex
        events  []*domain.ShippingEvent
        called  atomic.Int64
}

func (p *mockEventPublisher) Publish(_ context.Context, subject string, event *domain.ShippingEvent) error {
        p.mu.Lock()
        defer p.mu.Unlock()
        p.called.Add(1)
        p.events = append(p.events, event)
        return nil
}

func (p *mockEventPublisher) getEvents() []*domain.ShippingEvent {
        p.mu.Lock()
        defer p.mu.Unlock()
        cp := make([]*domain.ShippingEvent, len(p.events))
        copy(cp, p.events)
        return cp
}

// --- Helper to create a persisted shipment in a given status ---

func setupShipment(t *testing.T, repo *mockShipmentRepo, status domain.ShipmentStatus) *domain.Shipment {
        t.Helper()
        s := domain.NewShipment("order-1", "cust-1", domain.ShippingMethodStandard, "carrier-a", testAddress(), testPackages(), 599, "USD", 7)
        s.TrackingNumber = "TRK-123"
        s.ProviderRef = "PROV-abc123"
        repo.shipments[s.ID] = s

        // Walk through transitions to reach desired status
        transitions := map[domain.ShipmentStatus][]domain.ShipmentStatus{
                domain.ShipmentPending:    {},
                domain.ShipmentProcessing: {domain.ShipmentPending, domain.ShipmentProcessing},
                domain.ShipmentShipped:    {domain.ShipmentPending, domain.ShipmentProcessing, domain.ShipmentShipped},
                domain.ShipmentDelivered:  {domain.ShipmentPending, domain.ShipmentProcessing, domain.ShipmentShipped, domain.ShipmentDelivered},
        }

        steps, ok := transitions[status]
        require.True(t, ok, "invalid target status for setup: %s", status)
        for _, step := range steps {
                s.Status = step
                s.StatusHistory = append(s.StatusHistory, domain.StatusHistoryEntry{
                        Status: step, ChangedAt: time.Now().UTC(), Reason: "walk",
                })
        }
        repo.shipments[s.ID] = s
        return s
}

// --- Tests ---

func TestCalculateRate(t *testing.T) {
        ctx := context.Background()
        provider := shipInfra.NewMockShippingProvider()
        pub := &mockEventPublisher{}
        svc := NewService(nil, provider, pub)

        t.Run("returns rates for valid input", func(t *testing.T) {
                rates, err := svc.CalculateRate(ctx, RateRequestInput{
                        Destination: testAddress(),
                        Packages:    testPackages(),
                        Currency:    "USD",
                })
                require.NoError(t, err)
                require.NotEmpty(t, rates)
                assert.Equal(t, domain.ShippingMethodStandard, rates[0].Method)
                assert.Equal(t, "USD", rates[0].Currency)
                assert.Equal(t, provider.CalculateRateCalled.Load(), int64(1))
        })

        t.Run("defaults currency to BDT", func(t *testing.T) {
                rates, err := svc.CalculateRate(ctx, RateRequestInput{
                        Destination: testAddress(),
                        Packages:    testPackages(),
                })
                require.NoError(t, err)
                require.NotEmpty(t, rates)
                assert.Equal(t, "BDT", rates[0].Currency)
        })

        t.Run("errors on nil destination", func(t *testing.T) {
                _, err := svc.CalculateRate(ctx, RateRequestInput{
                        Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on empty packages", func(t *testing.T) {
                _, err := svc.CalculateRate(ctx, RateRequestInput{
                        Destination: testAddress(),
                        Packages:    []domain.ShipmentPackage{},
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on provider failure", func(t *testing.T) {
                p := shipInfra.NewMockShippingProvider()
                p.CalculateRateFn = func(_ context.Context, _ *domain.RateRequest) ([]*domain.ShippingRate, error) {
                        return nil, errors.New("PROVIDER_ERROR", "provider down", 502)
                }
                svc2 := NewService(nil, p, pub)
                _, err := svc2.CalculateRate(ctx, RateRequestInput{
                        Destination: testAddress(),
                        Packages:    testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 502, err.(*errors.AppError).Status)
        })
}

func TestCreateShipment(t *testing.T) {
        ctx := context.Background()
        provider := shipInfra.NewMockShippingProvider()
        pub := &mockEventPublisher{}

        t.Run("creates shipment and transitions to PROCESSING", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)

                input := CreateShipmentInput{
                        OrderID:     "order-1",
                        CustomerID:  "cust-1",
                        Method:      domain.ShippingMethodStandard,
                        Carrier:     "carrier-a",
                        Destination: testAddress(),
                        Packages:    testPackages(),
                        Currency:    "USD",
                }

                result, err := svc.CreateShipment(ctx, input)
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentProcessing, result.Status)
                assert.NotEmpty(t, result.TrackingNumber)
                assert.NotEmpty(t, result.ProviderRef)
                assert.Equal(t, "carrier-a", result.Carrier)
                assert.Equal(t, 599, result.CostCents)
                assert.Equal(t, 1, int(pub.called.Load()))

                events := pub.getEvents()
                assert.Equal(t, "shipment.created", events[0].EventType)
                assert.Equal(t, "PROCESSING", events[0].ToStatus)
                assert.Equal(t, "PENDING", events[0].FromStatus)
        })

        t.Run("errors on missing order_id", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        CustomerID:  "cust-1", Method: domain.ShippingMethodStandard,
                        Carrier: "carrier-a", Destination: testAddress(), Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on missing customer_id", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        OrderID: "order-1", Method: domain.ShippingMethodStandard,
                        Carrier: "carrier-a", Destination: testAddress(), Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on invalid method", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        OrderID: "order-1", CustomerID: "cust-1", Method: "INVALID",
                        Carrier: "carrier-a", Destination: testAddress(), Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on missing carrier", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        OrderID: "order-1", CustomerID: "cust-1", Method: domain.ShippingMethodStandard,
                        Carrier: "", Destination: testAddress(), Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on nil destination", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        OrderID: "order-1", CustomerID: "cust-1", Method: domain.ShippingMethodStandard,
                        Carrier: "carrier-a", Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("errors on empty packages", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        OrderID: "order-1", CustomerID: "cust-1", Method: domain.ShippingMethodStandard,
                        Carrier: "carrier-a", Destination: testAddress(), Packages: []domain.ShipmentPackage{},
                })
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("provider error returns 502", func(t *testing.T) {
                p := shipInfra.NewMockShippingProvider()
                p.CreateShipmentFn = func(_ context.Context, _ *domain.ProviderShipmentRequest) (*domain.ProviderShipmentResult, error) {
                        return nil, errors.New("PROVIDER_ERROR", "provider down", 502)
                }
                repo := newMockShipmentRepo()
                svc := NewService(repo, p, pub)
                _, err := svc.CreateShipment(ctx, CreateShipmentInput{
                        OrderID: "order-1", CustomerID: "cust-1", Method: domain.ShippingMethodStandard,
                        Carrier: "carrier-a", Destination: testAddress(), Packages: testPackages(),
                })
                require.Error(t, err)
                assert.Equal(t, 502, err.(*errors.AppError).Status)
        })

        t.Run("defaults currency to BDT", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                input := CreateShipmentInput{
                        OrderID: "order-1", CustomerID: "cust-1", Method: domain.ShippingMethodExpress,
                        Carrier: "carrier-a", Destination: testAddress(), Packages: testPackages(),
                }
                result, err := svc.CreateShipment(ctx, input)
                require.NoError(t, err)
                assert.Equal(t, "USD", result.Currency)
        })
}

func TestGetShipment(t *testing.T) {
        ctx := context.Background()
        repo := newMockShipmentRepo()
        svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)

        t.Run("finds by ID", func(t *testing.T) {
                s := setupShipment(t, repo, domain.ShipmentProcessing)
                found, err := svc.GetShipment(ctx, s.ID)
                require.NoError(t, err)
                assert.Equal(t, s.ID, found.ID)
                assert.Equal(t, s.ShipmentNumber, found.ShipmentNumber)
        })

        t.Run("not found returns error", func(t *testing.T) {
                _, err := svc.GetShipment(ctx, "nonexistent")
                require.Error(t, err)
                assert.Equal(t, 404, err.(*errors.AppError).Status)
        })
}

func TestGetShipmentByOrder(t *testing.T) {
        ctx := context.Background()
        repo := newMockShipmentRepo()
        svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)

        t.Run("finds by order ID", func(t *testing.T) {
                s := setupShipment(t, repo, domain.ShipmentProcessing)
                found, err := svc.GetShipmentByOrder(ctx, "order-1")
                require.NoError(t, err)
                assert.Equal(t, s.ID, found.ID)
        })

        t.Run("not found returns error", func(t *testing.T) {
                _, err := svc.GetShipmentByOrder(ctx, "no-such-order")
                require.Error(t, err)
                assert.Equal(t, 404, err.(*errors.AppError).Status)
        })

        t.Run("empty order_id returns validation error", func(t *testing.T) {
                _, err := svc.GetShipmentByOrder(ctx, "")
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })
}

func TestListShipments(t *testing.T) {
        ctx := context.Background()
        repo := newMockShipmentRepo()
        svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)

        setupShipment(t, repo, domain.ShipmentProcessing) // cust-1

        t.Run("lists customer shipments", func(t *testing.T) {
                shipments, total, err := svc.ListShipments(ctx, "cust-1", "", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.NoError(t, err)
                assert.Equal(t, int64(1), total)
                require.Len(t, shipments, 1)
        })

        t.Run("filters by status", func(t *testing.T) {
                shipments, total, err := svc.ListShipments(ctx, "cust-1", "PROCESSING", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.NoError(t, err)
                assert.Equal(t, int64(1), total)
                require.Len(t, shipments, 1)
        })

        t.Run("empty customer_id returns validation error", func(t *testing.T) {
                _, _, err := svc.ListShipments(ctx, "", "", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.Error(t, err)
                assert.Equal(t, 422, err.(*errors.AppError).Status)
        })

        t.Run("filter by non-matching status returns empty", func(t *testing.T) {
                shipments, total, err := svc.ListShipments(ctx, "cust-1", "DELIVERED", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.NoError(t, err)
                assert.Equal(t, int64(0), total)
                assert.Empty(t, shipments)
        })
}

func TestUpdateShipmentStatus(t *testing.T) {
        ctx := context.Background()

        t.Run("PROCESSING -> SHIPPED publishes event", func(t *testing.T) {
                pub := &mockEventPublisher{}
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), pub)
                s := setupShipment(t, repo, domain.ShipmentProcessing)

                updated, err := svc.UpdateShipmentStatus(ctx, s.ID, UpdateStatusInput{Status: "SHIPPED", Reason: "picked up"})
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentShipped, updated.Status)
                assert.Equal(t, 1, int(pub.called.Load()))
                events := pub.getEvents()
                assert.Equal(t, "shipment.shipped", events[0].EventType)
        })

        t.Run("SHIPPED -> DELIVERED publishes event", func(t *testing.T) {
                pub := &mockEventPublisher{}
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), pub)
                s := setupShipment(t, repo, domain.ShipmentShipped)

                updated, err := svc.UpdateShipmentStatus(ctx, s.ID, UpdateStatusInput{Status: "DELIVERED", Reason: "arrived"})
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentDelivered, updated.Status)
                assert.Equal(t, 1, int(pub.called.Load()))
                events := pub.getEvents()
                assert.Equal(t, "shipment.delivered", events[0].EventType)
        })

        t.Run("PENDING -> CANCELLED does not publish event (no transition event mapped)", func(t *testing.T) {
                pub := &mockEventPublisher{}
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), pub)
                s := setupShipment(t, repo, domain.ShipmentPending)

                updated, err := svc.UpdateShipmentStatus(ctx, s.ID, UpdateStatusInput{Status: "CANCELLED", Reason: "customer request"})
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentCancelled, updated.Status)
                assert.Equal(t, 0, int(pub.called.Load()))
        })

        t.Run("invalid transition returns error", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)
                s := setupShipment(t, repo, domain.ShipmentPending)

                _, err := svc.UpdateShipmentStatus(ctx, s.ID, UpdateStatusInput{Status: "DELIVERED", Reason: "skip"})
                require.Error(t, err)
                assert.Equal(t, 400, err.(*errors.AppError).Status)
        })

        t.Run("terminal state transition returns error", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)
                s := setupShipment(t, repo, domain.ShipmentDelivered)

                _, err := svc.UpdateShipmentStatus(ctx, s.ID, UpdateStatusInput{Status: "SHIPPED", Reason: "back"})
                require.Error(t, err)
                assert.Equal(t, 400, err.(*errors.AppError).Status)
        })

        t.Run("not found returns error", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)

                _, err := svc.UpdateShipmentStatus(ctx, "nonexistent", UpdateStatusInput{Status: "SHIPPED"})
                require.Error(t, err)
                assert.Equal(t, 404, err.(*errors.AppError).Status)
        })
}

func TestCancelShipment(t *testing.T) {
        ctx := context.Background()

        t.Run("cancels PENDING shipment", func(t *testing.T) {
                provider := shipInfra.NewMockShippingProvider()
                pub := &mockEventPublisher{}
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                s := setupShipment(t, repo, domain.ShipmentPending)

                result, err := svc.CancelShipment(ctx, s.ID, "customer request")
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentCancelled, result.Status)
                assert.Equal(t, 0, int(pub.called.Load())) // cancel has no event
                assert.Equal(t, int64(1), provider.CancelShipmentCalled.Load())
        })

        t.Run("cancels PROCESSING shipment", func(t *testing.T) {
                provider := shipInfra.NewMockShippingProvider()
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, nil)
                s := setupShipment(t, repo, domain.ShipmentProcessing)

                result, err := svc.CancelShipment(ctx, s.ID, "changed mind")
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentCancelled, result.Status)
                assert.Equal(t, int64(1), provider.CancelShipmentCalled.Load())
        })

        t.Run("cannot cancel DELIVERED shipment", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)
                s := setupShipment(t, repo, domain.ShipmentDelivered)

                _, err := svc.CancelShipment(ctx, s.ID, "oops")
                require.Error(t, err)
                assert.Equal(t, 400, err.(*errors.AppError).Status)
        })

        t.Run("defaults reason when empty", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)
                s := setupShipment(t, repo, domain.ShipmentPending)

                result, err := svc.CancelShipment(ctx, s.ID, "")
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentCancelled, result.Status)
        })

        t.Run("provider cancel error returns 502", func(t *testing.T) {
                p := shipInfra.NewMockShippingProvider()
                p.CancelShipmentFn = func(_ context.Context, _ string) error {
                        return errors.New("PROVIDER_ERROR", "cancel failed", 502)
                }
                repo := newMockShipmentRepo()
                svc := NewService(repo, p, nil)
                s := setupShipment(t, repo, domain.ShipmentProcessing)

                _, err := svc.CancelShipment(ctx, s.ID, "provider fail")
                require.Error(t, err)
                assert.Equal(t, 502, err.(*errors.AppError).Status)
        })
}

func TestTrackShipment(t *testing.T) {
        ctx := context.Background()

        t.Run("tracks shipment with provider ref", func(t *testing.T) {
                provider := shipInfra.NewMockShippingProvider()
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, nil)
                s := setupShipment(t, repo, domain.ShipmentProcessing)

                result, err := svc.TrackShipment(ctx, s.ID)
                require.NoError(t, err)
                assert.Equal(t, int64(1), provider.TrackShipmentCalled.Load())
                assert.NotEmpty(t, result.TrackingEvents)
        })

        t.Run("no provider ref returns shipment as-is", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)
                s := setupShipment(t, repo, domain.ShipmentProcessing)
                // Remove provider ref
                s.ProviderRef = ""
                repo.shipments[s.ID] = s

                result, err := svc.TrackShipment(ctx, s.ID)
                require.NoError(t, err)
                assert.Empty(t, result.TrackingEvents)
        })

        t.Run("not found returns error", func(t *testing.T) {
                repo := newMockShipmentRepo()
                svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)

                _, err := svc.TrackShipment(ctx, "nonexistent")
                require.Error(t, err)
                assert.Equal(t, 404, err.(*errors.AppError).Status)
        })

        t.Run("provider track error returns 502", func(t *testing.T) {
                p := shipInfra.NewMockShippingProvider()
                p.TrackShipmentFn = func(_ context.Context, _ string) ([]domain.TrackingEvent, error) {
                        return nil, errors.New("PROVIDER_ERROR", "track failed", 502)
                }
                repo := newMockShipmentRepo()
                svc := NewService(repo, p, nil)
                s := setupShipment(t, repo, domain.ShipmentProcessing)

                _, err := svc.TrackShipment(ctx, s.ID)
                require.Error(t, err)
                assert.Equal(t, 502, err.(*errors.AppError).Status)
        })

        t.Run("DELIVERED tracking event auto-transitions SHIPPED -> DELIVERED", func(t *testing.T) {
                provider := shipInfra.NewMockShippingProvider()
                pub := &mockEventPublisher{}
                provider.TrackShipmentFn = func(_ context.Context, _ string) ([]domain.TrackingEvent, error) {
                        return []domain.TrackingEvent{
                                {Status: "DELIVERED", Description: "Package delivered", Location: "Front door", Timestamp: time.Now().UTC()},
                        }, nil
                }
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                s := setupShipment(t, repo, domain.ShipmentShipped)

                result, err := svc.TrackShipment(ctx, s.ID)
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentDelivered, result.Status)
                events := pub.getEvents()
                require.NotEmpty(t, events)
                assert.Equal(t, "shipment.delivered", events[0].EventType)
        })

        t.Run("IN_TRANSIT event auto-transitions PROCESSING -> SHIPPED", func(t *testing.T) {
                provider := shipInfra.NewMockShippingProvider()
                pub := &mockEventPublisher{}
                provider.TrackShipmentFn = func(_ context.Context, _ string) ([]domain.TrackingEvent, error) {
                        return []domain.TrackingEvent{
                                {Status: "IN_TRANSIT", Description: "In transit", Location: "Distribution center", Timestamp: time.Now().UTC()},
                        }, nil
                }
                repo := newMockShipmentRepo()
                svc := NewService(repo, provider, pub)
                s := setupShipment(t, repo, domain.ShipmentProcessing)

                result, err := svc.TrackShipment(ctx, s.ID)
                require.NoError(t, err)
                assert.Equal(t, domain.ShipmentShipped, result.Status)
                events := pub.getEvents()
                require.NotEmpty(t, events)
                assert.Equal(t, "shipment.shipped", events[0].EventType)
        })
}

func TestDomainStatusTransitions(t *testing.T) {
        t.Run("all valid transitions", func(t *testing.T) {
                valid := map[domain.ShipmentStatus][]domain.ShipmentStatus{
                        domain.ShipmentPending:    {domain.ShipmentProcessing, domain.ShipmentCancelled},
                        domain.ShipmentProcessing: {domain.ShipmentShipped, domain.ShipmentCancelled},
                        domain.ShipmentShipped:    {domain.ShipmentDelivered},
                        domain.ShipmentDelivered:  {},
                        domain.ShipmentCancelled:  {},
                }
                for from, targets := range valid {
                        for _, to := range targets {
                                assert.True(t, from.CanTransitionTo(to), "expected %s -> %s to be valid", from, to)
                        }
                }
        })

        t.Run("invalid transitions fail", func(t *testing.T) {
                assert.False(t, domain.ShipmentPending.CanTransitionTo(domain.ShipmentShipped))
                assert.False(t, domain.ShipmentPending.CanTransitionTo(domain.ShipmentDelivered))
                assert.False(t, domain.ShipmentShipped.CanTransitionTo(domain.ShipmentCancelled))
                assert.False(t, domain.ShipmentDelivered.CanTransitionTo(domain.ShipmentShipped))
                assert.False(t, domain.ShipmentCancelled.CanTransitionTo(domain.ShipmentProcessing))
        })

        t.Run("terminal states", func(t *testing.T) {
                assert.True(t, domain.ShipmentDelivered.IsTerminal())
                assert.True(t, domain.ShipmentCancelled.IsTerminal())
                assert.False(t, domain.ShipmentPending.IsTerminal())
                assert.False(t, domain.ShipmentProcessing.IsTerminal())
                assert.False(t, domain.ShipmentShipped.IsTerminal())
        })

        t.Run("transition events mapping", func(t *testing.T) {
                assert.Equal(t, "shipment.created", domain.TransitionEvent(domain.ShipmentPending, domain.ShipmentProcessing))
                assert.Equal(t, "shipment.shipped", domain.TransitionEvent(domain.ShipmentProcessing, domain.ShipmentShipped))
                assert.Equal(t, "shipment.delivered", domain.TransitionEvent(domain.ShipmentShipped, domain.ShipmentDelivered))
                assert.Equal(t, "", domain.TransitionEvent(domain.ShipmentPending, domain.ShipmentCancelled))
        })

        t.Run("NewShipment creates in PENDING status", func(t *testing.T) {
                s := domain.NewShipment("order-1", "cust-1", domain.ShippingMethodStandard, "carrier-a", testAddress(), testPackages(), 599, "USD", 7)
                assert.Equal(t, domain.ShipmentPending, s.Status)
                assert.NotEmpty(t, s.ID)
                assert.Contains(t, s.ShipmentNumber, "SHP-")
                assert.Equal(t, 7, s.EstimatedDays)
                assert.NotNil(t, s.EstimatedDelivery)
                require.Len(t, s.StatusHistory, 1)
                assert.Equal(t, domain.ShipmentPending, s.StatusHistory[0].Status)
        })
}

func TestListAllShipments(t *testing.T) {
        ctx := context.Background()
        repo := newMockShipmentRepo()
        svc := NewService(repo, shipInfra.NewMockShippingProvider(), nil)

        setupShipment(t, repo, domain.ShipmentProcessing)
        setupShipment(t, repo, domain.ShipmentShipped)

        t.Run("lists all shipments", func(t *testing.T) {
                shipments, total, err := svc.ListAllShipments(ctx, "", "", "", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.NoError(t, err)
                assert.Equal(t, int64(2), total)
                assert.Len(t, shipments, 2)
        })

        t.Run("filters by carrier", func(t *testing.T) {
                _, total, err := svc.ListAllShipments(ctx, "", "", "carrier-a", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.NoError(t, err)
                assert.Equal(t, int64(2), total)
        })

        t.Run("filters by status", func(t *testing.T) {
                shipments, total, err := svc.ListAllShipments(ctx, "PROCESSING", "", "", pagination.Params{Page: 1, PerPage: 10, Skip: 0, Limit: 10})
                require.NoError(t, err)
                assert.Equal(t, int64(1), total)
                assert.Equal(t, domain.ShipmentProcessing, shipments[0].Status)
        })
}

func TestEstimatedDaysForMethod(t *testing.T) {
        assert.Equal(t, 7, estimatedDaysForMethod(domain.ShippingMethodStandard))
        assert.Equal(t, 3, estimatedDaysForMethod(domain.ShippingMethodExpress))
        assert.Equal(t, 1, estimatedDaysForMethod(domain.ShippingMethodOvernight))
        assert.Equal(t, 0, estimatedDaysForMethod(domain.ShippingMethodSameDay))
}

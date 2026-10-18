package orderworker

import (
        "context"

        ordDomain "github.com/ecommerce/backend/internal/orders/domain"
        payDomain "github.com/ecommerce/backend/internal/payments/domain"
        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/logger"
)

// OrderGetter retrieves an order by ID.
type OrderGetter interface {
        GetOrderByID(ctx context.Context, id string) (*ordDomain.Order, error)
}

// OrderStatusUpdater transitions an order to a new status.
type OrderStatusUpdater interface {
        UpdateOrderStatus(ctx context.Context, orderID, status, reason, paymentReference string) (*ordDomain.Order, error)
}

// ReservationReleaser releases all active reservations for a reference.
type ReservationReleaser interface {
        ReleaseByReference(ctx context.Context, reference string) (int, error)
}

// HandlerConfig holds the dependencies for the order worker handlers.
type HandlerConfig struct {
        Orders    OrderGetter
        Updaters  OrderStatusUpdater
        Inventory ReservationReleaser
        Log       *logger.Logger
}

// PaymentCompletedHandler returns an EventHandler that processes payment.completed events.
// Flow: validate event -> load order -> validate order state (PAYMENT_PENDING) ->
//        mark order PAID with payment reference -> publish order.paid.
func PaymentCompletedHandler(cfg HandlerConfig) eventstore.EventHandler {
        return func(ctx context.Context, env *eventstore.Envelope) error {
                var evt payDomain.PaymentEvent
                if err := env.DecodePayload(&evt); err != nil {
                        cfg.Log.Errorw("failed to decode payment.completed payload",
                                "event_id", env.EventID, "error", err)
                        return nil // non-retryable: malformed payload won't fix itself
                }

                if evt.OrderID == "" {
                        cfg.Log.Errorw("payment.completed missing order_id",
                                "event_id", env.EventID, "payment_id", evt.PaymentID)
                        return nil
                }

                cfg.Log.Infow("processing payment.completed",
                        "event_id", env.EventID,
                        "payment_id", evt.PaymentID,
                        "order_id", evt.OrderID,
                        "provider_ref", evt.ProviderRef,
                )

                // Load order and validate state.
                order, err := cfg.Orders.GetOrderByID(ctx, evt.OrderID)
                if err != nil {
                        cfg.Log.Errorw("failed to find order for payment.completed",
                                "event_id", env.EventID,
                                "order_id", evt.OrderID,
                                "error", err,
                        )
                        return err // retryable: order might not be replicated yet
                }

                // Idempotency: if order is already PAID or beyond, this is a duplicate.
                if order.Status != ordDomain.OrderPaymentPending {
                        cfg.Log.Infow("skipping payment.completed: order not in PAYMENT_PENDING",
                                "event_id", env.EventID,
                                "order_id", order.ID,
                                "order_status", order.Status,
                        )
                        return nil // not an error — just a duplicate
                }

                // Transition to PAID.
                _, err = cfg.Updaters.UpdateOrderStatus(ctx, order.ID,
                        string(ordDomain.OrderPaid), "payment completed", evt.ProviderRef)
                if err != nil {
                        cfg.Log.Errorw("failed to transition order to PAID",
                                "event_id", env.EventID,
                                "order_id", order.ID,
                                "error", err,
                        )
                        return err // retryable: might be a transient conflict
                }

                cfg.Log.Infow("order marked PAID after payment.completed",
                        "event_id", env.EventID,
                        "order_id", order.ID,
                        "payment_id", evt.PaymentID,
                )

                return nil
        }
}

// PaymentFailedHandler returns an EventHandler that processes payment.failed events.
// Flow: validate event -> load order -> validate order state ->
//        cancel order -> release inventory reservations -> publish order.cancelled.
func PaymentFailedHandler(cfg HandlerConfig) eventstore.EventHandler {
        return func(ctx context.Context, env *eventstore.Envelope) error {
                var evt payDomain.PaymentEvent
                if err := env.DecodePayload(&evt); err != nil {
                        cfg.Log.Errorw("failed to decode payment.failed payload",
                                "event_id", env.EventID, "error", err)
                        return nil
                }

                if evt.OrderID == "" {
                        cfg.Log.Errorw("payment.failed missing order_id",
                                "event_id", env.EventID, "payment_id", evt.PaymentID)
                        return nil
                }

                cfg.Log.Infow("processing payment.failed",
                        "event_id", env.EventID,
                        "payment_id", evt.PaymentID,
                        "order_id", evt.OrderID,
                )

                // Load order.
                order, err := cfg.Orders.GetOrderByID(ctx, evt.OrderID)
                if err != nil {
                        cfg.Log.Errorw("failed to find order for payment.failed",
                                "event_id", env.EventID,
                                "order_id", evt.OrderID,
                                "error", err,
                        )
                        return err
                }

                // Idempotency: if order is already CANCELLED, REFUNDED, or beyond, skip.
                if order.Status == ordDomain.OrderCancelled ||
                        order.Status == ordDomain.OrderRefunded {
                        cfg.Log.Infow("skipping payment.failed: order already in terminal/cancelled state",
                                "event_id", env.EventID,
                                "order_id", order.ID,
                                "order_status", order.Status,
                        )
                        return nil
                }

                // Cancel the order.
                _, err = cfg.Updaters.UpdateOrderStatus(ctx, order.ID,
                        string(ordDomain.OrderCancelled), "payment failed: "+evt.ProviderRef, "")
                if err != nil {
                        cfg.Log.Errorw("failed to cancel order after payment failure",
                                "event_id", env.EventID,
                                "order_id", order.ID,
                                "error", err,
                        )
                        return err
                }

                // Release all inventory reservations for this order.
                released, relErr := cfg.Inventory.ReleaseByReference(ctx, evt.OrderID)
                if relErr != nil {
                        cfg.Log.Warnw("failed to release inventory reservations",
                                "event_id", env.EventID,
                                "order_id", evt.OrderID,
                                "error", relErr,
                        )
                        // Don't return error — order is already cancelled, reservations will expire
                } else if released > 0 {
                        cfg.Log.Infow("released inventory reservations after payment failure",
                                "event_id", env.EventID,
                                "order_id", evt.OrderID,
                                "released_count", released,
                        )
                }

                cfg.Log.Infow("order cancelled after payment.failed",
                        "event_id", env.EventID,
                        "order_id", order.ID,
                        "payment_id", evt.PaymentID,
                )

                return nil
        }
}

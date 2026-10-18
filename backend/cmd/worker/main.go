package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	catInfra "github.com/ecommerce/backend/internal/categories/infrastructure"
	custInfra "github.com/ecommerce/backend/internal/customers/infrastructure"
	invApp "github.com/ecommerce/backend/internal/inventory/application"
	invInfra "github.com/ecommerce/backend/internal/inventory/infrastructure"
	mktInfra "github.com/ecommerce/backend/internal/marketing/infrastructure"
	mktWorker "github.com/ecommerce/backend/internal/marketing/worker"
	notifDomain "github.com/ecommerce/backend/internal/notifications/domain"
	notifInfra "github.com/ecommerce/backend/internal/notifications/infrastructure"
	notifWorker "github.com/ecommerce/backend/internal/notifications/worker"
	ordApp "github.com/ecommerce/backend/internal/orders/application"
	ordDomain "github.com/ecommerce/backend/internal/orders/domain"
	ordInfra "github.com/ecommerce/backend/internal/orders/infrastructure"
	payInfra "github.com/ecommerce/backend/internal/payments/infrastructure"
	userInfra "github.com/ecommerce/backend/internal/users/infrastructure"
	"github.com/ecommerce/backend/pkg/config"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/emailtemplate"
	"github.com/ecommerce/backend/pkg/mailer"
	"github.com/ecommerce/backend/pkg/logger"
	"github.com/ecommerce/backend/pkg/mongodb"
	"github.com/ecommerce/backend/pkg/nats"
	"github.com/ecommerce/backend/pkg/observability"
	"github.com/ecommerce/backend/pkg/orderworker"
	"github.com/ecommerce/backend/pkg/outbox"
	"github.com/ecommerce/backend/pkg/telemetry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.Logging.Level, cfg.Logging.Encoding)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Infow("starting ecommerce worker", "env", os.Getenv("APP_ENV"))

	// Initialize OpenTelemetry (Prometheus metrics + stdout traces)
	var (
		obsProvider *observability.Provider
		allMetrics  *telemetry.AllMetrics
	)
	if cfg.Observability.Enabled {
		var err error
		obsProvider, err = observability.NewProvider(cfg.Observability.ServiceName+"-worker", cfg.Observability.PrometheusPort)
		if err != nil {
			log.Fatalw("failed to initialize observability", "error", err)
		}
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = obsProvider.Shutdown(ctx)
		}()

		allMetrics, err = telemetry.NewAllMetrics(obsProvider.Meter("ecommerce-worker"))
		if err != nil {
			log.Fatalw("failed to initialize metrics", "error", err)
		}
		log.Infow("observability initialized", "prometheus_port", cfg.Observability.PrometheusPort)
	}

	// Connect to MongoDB
	mongoClient, err := mongodb.New(
		ctx,
		cfg.MongoDB.URI,
		cfg.MongoDB.Database,
		cfg.MongoDB.MaxPoolSize,
		cfg.MongoDB.MinPoolSize,
		cfg.MongoDB.ConnectTimeout,
	)
	if err != nil {
		log.Fatalw("failed to connect to mongodb", "error", err)
	}
	defer mongoClient.Close(ctx)

	// Ensure indexes
	db := mongoClient.Database()
	if err := catInfra.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create category indexes", "error", err)
	}
	if err := invInfra.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create inventory indexes", "error", err)
	}
	if err := ordInfra.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create order indexes", "error", err)
	}
	if err := payInfra.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create payment indexes", "error", err)
	}
	if err := payInfra.EnsureWebhookIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create webhook event indexes", "error", err)
	}
	if err := outbox.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create outbox indexes", "error", err)
	}
	if err := notifInfra.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create notification indexes", "error", err)
	}
	if err := mktInfra.EnsureIndexes(ctx, db); err != nil {
		log.Fatalw("failed to create marketing indexes", "error", err)
	}
	log.Info("mongodb indexes ensured")

	// Connect to NATS
	natsClient, err := nats.New(
		ctx,
		cfg.NATS.URL,
		cfg.NATS.ClusterName,
		cfg.NATS.ReconnectWait,
		cfg.NATS.MaxReconnects,
		cfg.NATS.ConnectTimeout,
	)
	if err != nil {
		log.Fatalw("failed to connect to nats", "error", err)
	}
	defer natsClient.Close()

	// Ensure JetStream event streams
	streamMgr := eventstore.NewStreamManager(natsClient.JS())
	if err := streamMgr.EnsureStreams(ctx, eventstore.DefaultStreamConfigs()); err != nil {
		log.Fatalw("failed to ensure event streams", "error", err)
	}
	log.Info("jetstream streams ensured")

	// --- Wire up outbox worker ---
	outboxRepo := outbox.NewOutboxRepository(db)
	eventstorePublisher := eventstore.NewPublisher(natsClient.JS(), log)

	// Wire NATS metrics and tracing into the event publisher.
	if allMetrics != nil {
		eventstorePublisher.SetNATSMetrics(allMetrics.NATS)
	}
	natsTracer := telemetry.NewNATSTracer("github.com/ecommerce/backend/pkg/eventstore")
	eventstorePublisher.SetNATSTracer(natsTracer)

	outboxWorker := outbox.NewWorker(outboxRepo, eventstorePublisher, log, outbox.DefaultWorkerConfig())
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()
	log.Info("outbox worker started")

	// --- Wire up outbox-backed publishers ---
	orderOutboxPublisher := outbox.NewOrderPublisherAdapter(outboxRepo)
	inventoryOutboxPublisher := outbox.NewInventoryPublisherAdapter(outboxRepo)

	// --- Wire up order service ---
	orderRepo := ordInfra.NewOrderRepository(db)
	orderService := ordApp.NewService(orderRepo, orderOutboxPublisher)

	// --- Wire up inventory service ---
	stockRepo := invInfra.NewStockRepository(db)
	reservationRepo := invInfra.NewReservationRepository(db)
	movementRepo := invInfra.NewMovementRepository(db)
	warehouseRepo := invInfra.NewWarehouseRepository(db)
	inventoryService := invApp.NewService(stockRepo, reservationRepo, movementRepo, warehouseRepo, inventoryOutboxPublisher)
	// Online orders take their stock when the payment completes (handled by the order service).
	orderService.SetStock(ordInfra.NewStockReserver(inventoryService))
	orderService.SetCoupons(ordInfra.NewCouponApplier(db))

	// --- Helper to wire NATS metrics into a subscriber ---
	wireNATSMetrics := func(sub *eventstore.Subscriber) {
		if allMetrics != nil {
			sub.SetNATSMetrics(allMetrics.NATS)
		}
	}

	// --- Wire up order worker event subscriptions ---
	handlerCfg := orderworker.HandlerConfig{
		Orders:    orderService,
		Updaters:  &orderStatusUpdaterAdapter{svc: orderService},
		Inventory: inventoryService,
		Log:       log,
	}

	completedSub := eventstore.NewSubscriber(natsClient.JS(), log, orderworker.PaymentCompletedHandler(handlerCfg))
	wireNATSMetrics(completedSub)

	failedSub := eventstore.NewSubscriber(natsClient.JS(), log, orderworker.PaymentFailedHandler(handlerCfg))
	wireNATSMetrics(failedSub)

	// Launch order worker subscribers in goroutines (Subscribe blocks until ctx cancelled).
	go func() {
		log.Infow("starting payment.completed subscriber")
		if err := completedSub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamPayments,
			ConsumerName:  "order-worker-payment-completed",
			FilterSubject: "ECOMMERCE_PAYMENTS.payment.completed",
			Description:   "Transitions order to PAID on payment completion",
		}); err != nil {
			log.Errorw("payment.completed subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting payment.failed subscriber")
		if err := failedSub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamPayments,
			ConsumerName:  "order-worker-payment-failed",
			FilterSubject: "ECOMMERCE_PAYMENTS.payment.failed",
			Description:   "Cancels order and releases inventory on payment failure",
		}); err != nil {
			log.Errorw("payment.failed subscriber stopped", "error", err)
		}
	}()

	// --- Wire up notification worker ---
	notifRepo := notifInfra.NewNotificationRepository(db)
	userRepo := userInfra.NewUserRepository(db)
	customerRepo := custInfra.NewCustomerRepository(db)
	contactResolver := notifWorker.NewOrderCustomerResolver(userRepo, customerRepo)

	// Emails go out through SMTP (Mailpit in development, your provider in production).
	// SMS and push are still log-only.
	var emailProvider notifDomain.EmailProvider = notifInfra.NewSMTPEmailProvider(
		mailer.NewSMTP(mailer.SMTPConfig{
			Host: cfg.Email.SMTPHost, Port: cfg.Email.SMTPPort,
			Username: cfg.Email.SMTPUsername, Password: cfg.Email.SMTPPassword,
			From: cfg.Email.SMTPFrom, StartTLS: cfg.Email.SMTPStartTLS,
			Timeout: cfg.Email.SMTPTimeout,
		}),
		emailtemplate.Brand{
			Name:         envOr("APP_NAME", "FarmNest"),
			LogoURL:      os.Getenv("BRAND_LOGO_URL"),
			Color:        envOr("BRAND_COLOR", "#1f5c3a"),
			AppURL:       envOr("APP_URL", "http://localhost:5174"),
			AdminURL:     envOr("ADMIN_URL", "http://localhost:5173"),
			SupportEmail: os.Getenv("SUPPORT_EMAIL"),
		},
	)
	smsProvider := notifInfra.NewLogSMSProvider()
	pushProvider := notifInfra.NewLogPushProvider()

	notifDeps := notifWorker.DispatcherDependencies{
		EmailProvider:    emailProvider,
		SMSProvider:      smsProvider,
		PushProvider:     pushProvider,
		NotificationRepo: notifRepo,
		ContactResolver:  contactResolver,
		Log:              log,
		Config:           notifWorker.DefaultDispatcherConfig(),
	}
	notifDispatcher := notifWorker.NewDispatcher(notifDeps)

	// Notification subscribers — durable consumers on ECOMMERCE_ORDERS and ECOMMERCE_PAYMENTS.
	// These never block HTTP requests; they run as separate JetStream consumers.
	go func() {
		log.Infow("starting notification: order.created subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, notifDispatcher.HandleOrderCreated())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamOrders,
			ConsumerName:  "notification-order-created",
			FilterSubject: "ECOMMERCE_ORDERS.order.created",
			Description:   "Sends order confirmation notifications",
		}); err != nil {
			log.Errorw("notification: order.created subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting notification: order.paid subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, notifDispatcher.HandleOrderPaid())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamOrders,
			ConsumerName:  "notification-order-paid",
			FilterSubject: "ECOMMERCE_ORDERS.order.paid",
			Description:   "Sends payment confirmation notifications",
		}); err != nil {
			log.Errorw("notification: order.paid subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting notification: order.shipped subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, notifDispatcher.HandleOrderShipped())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamOrders,
			ConsumerName:  "notification-order-shipped",
			FilterSubject: "ECOMMERCE_ORDERS.order.shipped",
			Description:   "Sends shipping confirmation notifications",
		}); err != nil {
			log.Errorw("notification: order.shipped subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting notification: order.delivered subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, notifDispatcher.HandleOrderDelivered())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamOrders,
			ConsumerName:  "notification-order-delivered",
			FilterSubject: "ECOMMERCE_ORDERS.order.delivered",
			Description:   "Sends delivery confirmation notifications",
		}); err != nil {
			log.Errorw("notification: order.delivered subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting notification: payment.failed subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, notifDispatcher.HandlePaymentFailed())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamPayments,
			ConsumerName:  "notification-payment-failed",
			FilterSubject: "ECOMMERCE_PAYMENTS.payment.failed",
			Description:   "Sends payment failure notifications to customers",
		}); err != nil {
			log.Errorw("notification: payment.failed subscriber stopped", "error", err)
		}
	}()

	// --- Wire up marketing analytics processor ---
	mktEventRepo := mktInfra.NewAnalyticsEventRepository(db)
	mktCampaignRepo := mktInfra.NewCampaignRepository(db)
	mktSegmentRepo := mktInfra.NewSegmentRepository(db)
	mktPublisher := outbox.NewMarketingPublisherAdapter(outboxRepo)

	mktProcessor := mktWorker.NewProcessor(mktWorker.ProcessorDependencies{
		EventRepo:      mktEventRepo,
		CampaignRepo:   mktCampaignRepo,
		SegmentRepo:    mktSegmentRepo,
		EventPublisher: mktPublisher,
		Log:            log,
		Config:         mktWorker.DefaultProcessorConfig(),
	})

	// Marketing analytics subscribers
	go func() {
		log.Infow("starting marketing: analytics.product_viewed subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, mktProcessor.HandleProductViewed())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamAnalytics,
			ConsumerName:  "marketing-product-viewed",
			FilterSubject: "ECOMMERCE_ANALYTICS.analytics.product_viewed",
			Description:   "Processes product view events for marketing",
		}); err != nil {
			log.Errorw("marketing: product_viewed subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting marketing: analytics.cart_created subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, mktProcessor.HandleCartCreated())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamAnalytics,
			ConsumerName:  "marketing-cart-created",
			FilterSubject: "ECOMMERCE_ANALYTICS.analytics.cart_created",
			Description:   "Processes cart created events for marketing",
		}); err != nil {
			log.Errorw("marketing: cart_created subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting marketing: analytics.cart_abandoned subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, mktProcessor.HandleCartAbandoned())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamAnalytics,
			ConsumerName:  "marketing-cart-abandoned",
			FilterSubject: "ECOMMERCE_ANALYTICS.analytics.cart_abandoned",
			Description:   "Processes abandoned cart events for marketing",
		}); err != nil {
			log.Errorw("marketing: cart_abandoned subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting marketing: analytics.checkout_started subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, mktProcessor.HandleCheckoutStarted())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamAnalytics,
			ConsumerName:  "marketing-checkout-started",
			FilterSubject: "ECOMMERCE_ANALYTICS.analytics.checkout_started",
			Description:   "Processes checkout started events for marketing",
		}); err != nil {
			log.Errorw("marketing: checkout_started subscriber stopped", "error", err)
		}
	}()

	go func() {
		log.Infow("starting marketing: analytics.order_completed subscriber")
		sub := eventstore.NewSubscriber(natsClient.JS(), log, mktProcessor.HandleOrderCompleted())
		wireNATSMetrics(sub)
		if err := sub.Subscribe(ctx, eventstore.SubscriberOptions{
			Stream:        eventstore.StreamAnalytics,
			ConsumerName:  "marketing-order-completed",
			FilterSubject: "ECOMMERCE_ANALYTICS.analytics.order_completed",
			Description:   "Processes order completed events for marketing",
		}); err != nil {
			log.Errorw("marketing: order_completed subscriber stopped", "error", err)
		}
	}()

	log.Info("worker initialized — subscriptions active")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Infow("shutting down worker", "signal", sig.String())

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	log.Info("worker stopped")
}

// orderStatusUpdaterAdapter adapts the order application service to the worker's OrderStatusUpdater interface.
type orderStatusUpdaterAdapter struct {
	svc *ordApp.Service
}

func (a *orderStatusUpdaterAdapter) UpdateOrderStatus(ctx context.Context, id, status, reason, payRef string) (*ordDomain.Order, error) {
	return a.svc.UpdateOrderStatus(ctx, id, ordApp.UpdateStatusInput{
		Status:           status,
		Reason:           reason,
		PaymentReference: payRef,
	})
}
// envOr returns the environment variable, or the fallback when it is empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package main

import (
	stderrors "errors"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ecommerce/backend/internal/auth/application"
	authInfra "github.com/ecommerce/backend/internal/auth/infrastructure"
	authTransport "github.com/ecommerce/backend/internal/auth/transport"
	catApp "github.com/ecommerce/backend/internal/categories/application"
	catInfra "github.com/ecommerce/backend/internal/categories/infrastructure"
	catTransport "github.com/ecommerce/backend/internal/categories/transport"
	customerInfra "github.com/ecommerce/backend/internal/customers/infrastructure"
	"github.com/ecommerce/backend/internal/health"

	adminApp "github.com/ecommerce/backend/internal/admin/application"
	adminTransport "github.com/ecommerce/backend/internal/admin/transport"
	analyticsApp "github.com/ecommerce/backend/internal/analytics/application"
	analyticsInfra "github.com/ecommerce/backend/internal/analytics/infrastructure"
	analyticsTransport "github.com/ecommerce/backend/internal/analytics/transport"
	analyticsWorker "github.com/ecommerce/backend/internal/analytics/worker"
	invApp "github.com/ecommerce/backend/internal/inventory/application"
	invInfra "github.com/ecommerce/backend/internal/inventory/infrastructure"
	invTransport "github.com/ecommerce/backend/internal/inventory/transport"
	mktApp "github.com/ecommerce/backend/internal/marketing/application"
	mktInfra "github.com/ecommerce/backend/internal/marketing/infrastructure"
	mktTransport "github.com/ecommerce/backend/internal/marketing/transport"
	"github.com/ecommerce/backend/internal/middleware"
	ordApp "github.com/ecommerce/backend/internal/orders/application"
	ordInfra "github.com/ecommerce/backend/internal/orders/infrastructure"
	ordTransport "github.com/ecommerce/backend/internal/orders/transport"
	payApp "github.com/ecommerce/backend/internal/payments/application"
	payInfra "github.com/ecommerce/backend/internal/payments/infrastructure"
	payTransport "github.com/ecommerce/backend/internal/payments/transport"
	prodApp "github.com/ecommerce/backend/internal/products/application"
	setApp "github.com/ecommerce/backend/internal/settings/application"
	setInfra "github.com/ecommerce/backend/internal/settings/infrastructure"
	setTransport "github.com/ecommerce/backend/internal/settings/transport"
	supApp "github.com/ecommerce/backend/internal/support/application"
	supInfra "github.com/ecommerce/backend/internal/support/infrastructure"
	supTransport "github.com/ecommerce/backend/internal/support/transport"
	repInfra "github.com/ecommerce/backend/internal/reports/infrastructure"
	repTransport "github.com/ecommerce/backend/internal/reports/transport"
	purApp "github.com/ecommerce/backend/internal/purchases/application"
	purInfra "github.com/ecommerce/backend/internal/purchases/infrastructure"
	purTransport "github.com/ecommerce/backend/internal/purchases/transport"
	vendApp "github.com/ecommerce/backend/internal/vendors/application"
	vendInfra "github.com/ecommerce/backend/internal/vendors/infrastructure"
	vendTransport "github.com/ecommerce/backend/internal/vendors/transport"
	prodInfra "github.com/ecommerce/backend/internal/products/infrastructure"
	prodTransport "github.com/ecommerce/backend/internal/products/transport"
	revApp "github.com/ecommerce/backend/internal/reviews/application"
	revInfra "github.com/ecommerce/backend/internal/reviews/infrastructure"
	revTransport "github.com/ecommerce/backend/internal/reviews/transport"
	shipApp "github.com/ecommerce/backend/internal/shipping/application"
	shipInfra "github.com/ecommerce/backend/internal/shipping/infrastructure"
	shipTransport "github.com/ecommerce/backend/internal/shipping/transport"
	userDomain "github.com/ecommerce/backend/internal/users/domain"
	userInfra "github.com/ecommerce/backend/internal/users/infrastructure"
	userTransport "github.com/ecommerce/backend/internal/users/transport"
	wishApp "github.com/ecommerce/backend/internal/wishlist/application"
	wishInfra "github.com/ecommerce/backend/internal/wishlist/infrastructure"
	wishTransport "github.com/ecommerce/backend/internal/wishlist/transport"
	"github.com/ecommerce/backend/pkg/audit"
	"github.com/ecommerce/backend/pkg/config"
	"github.com/ecommerce/backend/pkg/eventstore"
	"github.com/ecommerce/backend/pkg/logger"
	"github.com/ecommerce/backend/pkg/emailtemplate"
	"github.com/ecommerce/backend/pkg/mailer"
	"github.com/ecommerce/backend/pkg/mongodb"
	"github.com/ecommerce/backend/pkg/nats"
	"github.com/ecommerce/backend/pkg/observability"
	"github.com/ecommerce/backend/pkg/outbox"
	"github.com/ecommerce/backend/pkg/telemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
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

        log.Infow("starting ecommerce api", "port", cfg.Server.Port, "env", os.Getenv("APP_ENV"))

        // Initialize OpenTelemetry (Prometheus metrics + stdout traces)
        var (
                obsProvider *observability.Provider
                allMetrics  *telemetry.AllMetrics
        )
        if cfg.Observability.Enabled {
                var err error
                obsProvider, err = observability.NewProvider(cfg.Observability.ServiceName, cfg.Observability.PrometheusPort)
                if err != nil {
                        log.Fatalw("failed to initialize observability", "error", err)
                }
                defer func() {
                        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
                        defer cancel()
                        _ = obsProvider.Shutdown(ctx)
                }()

                allMetrics, err = telemetry.NewAllMetrics(obsProvider.Meter("ecommerce-backend"))
                if err != nil {
                        log.Fatalw("failed to initialize metrics", "error", err)
                }
                log.Infow("observability initialized", "prometheus_port", cfg.Observability.PrometheusPort)
        }

        // Initialize MongoDB
        mongoClient, err := mongodb.New(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.MaxPoolSize, cfg.MongoDB.MinPoolSize, cfg.MongoDB.ConnectTimeout)
        if err != nil {
                log.Fatalw("failed to connect to mongodb", "error", err)
        }
        defer mongoClient.Close(ctx)
        log.Info("mongodb connected")

        // Ensure indexes
        db := mongoClient.Database()
        if err := userInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create user indexes", "error", err)
        }
        if err := authInfra.EnsureTokenIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create token indexes", "error", err)
        }
        if err := customerInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create customer indexes", "error", err)
        }
        if err := supInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to ensure support indexes", "error", err)
        }
        if err := purInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to ensure purchase indexes", "error", err)
        }
        if err := vendInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to ensure vendor indexes", "error", err)
        }
        if err := prodInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create product indexes", "error", err)
        }
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
        if err := shipInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create shipment indexes", "error", err)
        }
        if err := revInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create review indexes", "error", err)
        }
        if err := wishInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create wishlist indexes", "error", err)
        }
        if err := mktInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create marketing indexes", "error", err)
        }
        if err := analyticsInfra.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create analytics indexes", "error", err)
        }
        if err := payInfra.EnsureWebhookIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create webhook event indexes", "error", err)
        }
        // Outbox must be initialized BEFORE any outbox-backed publisher is used.
        if err := outbox.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create outbox indexes", "error", err)
        }
        if err := audit.EnsureIndexes(ctx, db); err != nil {
                log.Fatalw("failed to create audit indexes", "error", err)
        }
        log.Info("mongodb indexes ensured")

        // Initialize NATS
        natsClient, err := nats.New(ctx, cfg.NATS.URL, cfg.NATS.ClusterName, cfg.NATS.ReconnectWait, cfg.NATS.MaxReconnects, cfg.NATS.ConnectTimeout)
        if err != nil {
                log.Fatalw("failed to connect to nats", "error", err)
        }
        defer natsClient.Close()
        log.Info("nats connected")

        // Ensure JetStream event streams
        streamMgr := eventstore.NewStreamManager(natsClient.JS())
        if err := streamMgr.EnsureStreams(ctx, eventstore.DefaultStreamConfigs()); err != nil {
                log.Fatalw("failed to ensure event streams", "error", err)
        }
        log.Info("jetstream streams ensured")

        // Initialize KV buckets
        requiredBuckets := make([]string, 0, len(cfg.NATSKV.Buckets))
        for _, bucketCfg := range cfg.NATSKV.Buckets {
                _, err := natsClient.InitKVBucket(ctx, bucketCfg.Name, bucketCfg.History)
                if err != nil {
                        log.Fatalw("failed to initialize KV bucket", "bucket", bucketCfg.Name, "error", err)
                }
                requiredBuckets = append(requiredBuckets, bucketCfg.Name)
        }

        // --- Wire up auth ---
        jwtService := authInfra.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL, cfg.JWT.Issuer)
        passwordHasher := authInfra.NewPasswordHasher(12)

        userRepo := userInfra.NewUserRepository(db)
        tokenRepo := authInfra.NewTokenRepository(db)
        customerRepo := customerInfra.NewCustomerRepository(db)
        emailSender := mailer.NewSMTP(mailer.SMTPConfig{
                Host: cfg.Email.SMTPHost, Port: cfg.Email.SMTPPort,
                Username: cfg.Email.SMTPUsername, Password: cfg.Email.SMTPPassword,
                From: cfg.Email.SMTPFrom, StartTLS: cfg.Email.SMTPStartTLS,
                Timeout: cfg.Email.SMTPTimeout,
        })

        authService := application.NewService(
                userRepo,
                tokenRepo,
                customerRepo,
                jwtService,
                passwordHasher,
                application.DefaultAuthConfig(),
                cfg.JWT.RefreshTTL,
                emailSender,
        )

        // Branded emails: the name, logo and the addresses their links point to.
        brand := emailtemplate.Brand{
                Name:         envOr("APP_NAME", "FarmNest"),
                LogoURL:      os.Getenv("BRAND_LOGO_URL"),
                Color:        envOr("BRAND_COLOR", "#1f5c3a"),
                AppURL:       envOr("APP_URL", "http://localhost:5174"),
                AdminURL:     envOr("ADMIN_URL", "http://localhost:5173"),
                SupportEmail: os.Getenv("SUPPORT_EMAIL"),
        }
        authService.SetBrand(brand)

        authHandler := authTransport.NewHandler(authService)
        userHandler := userTransport.NewHandler(userRepo)
        authMW := authTransport.AuthMiddleware(authService)
        adminMW := authTransport.RequireRole(userDomain.RoleSuperAdmin, userDomain.RoleAdmin, userDomain.RoleManager)

        // --- Wire up shared audit auditor (async, fire-and-forget) ---
        auditRepo := audit.NewRepository(db)
        auditor := audit.NewAuditor(auditRepo, log)
        defer auditor.Stop()
        log.Info("audit auditor started")

        // Inject auditor into user handler for profile change auditing
        userHandler.SetAuditor(auditor)

        // --- Wire up outbox (transactional event publishing) ---
        // MUST be created before any outbox-backed publisher adapter is used.
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

        // Outbox-backed event publishers for each domain.
        // These replace direct NATS publishers to guarantee at-least-once delivery.
        orderOutboxPublisher := outbox.NewOrderPublisherAdapter(outboxRepo)
        inventoryOutboxPublisher := outbox.NewInventoryPublisherAdapter(outboxRepo)
        paymentOutboxPublisher := outbox.NewPaymentPublisherAdapter(outboxRepo)
        shippingOutboxPublisher := outbox.NewShippingPublisherAdapter(outboxRepo)

        // --- Wire up products & categories ---
        productKV, err := natsClient.KV("PRODUCT_CACHE")
        if err != nil {
                log.Fatalw("failed to get PRODUCT_CACHE KV bucket", "error", err)
        }
        categoryKV, err := natsClient.KV("CATEGORY_CACHE")
        if err != nil {
                log.Fatalw("failed to get CATEGORY_CACHE KV bucket", "error", err)
        }

        productRepo := prodInfra.NewProductRepository(db)
        productCache := prodInfra.NewProductCache(productKV)
        productService := prodApp.NewService(productRepo, productCache)
        productService.SetAuditor(auditor)
        productService.SetStockChecker(prodInfra.NewStockChecker(db))
        productService.SetRatingReader(prodInfra.NewRatingReader(db))
        productService.SetEvents(outbox.NewProductPublisherAdapter(outboxRepo))
        productHandler := prodTransport.NewHandler(productService)

        categoryRepo := catInfra.NewCategoryRepository(db)
        categoryCache := catInfra.NewCategoryCache(categoryKV)
        categoryService := catApp.NewService(categoryRepo, categoryCache)
        categoryService.SetAuditor(auditor)
        categoryHandler := catTransport.NewHandler(categoryService)

        // --- Wire up shipping ---
        shipmentRepo := shipInfra.NewShipmentRepository(db)
        shippingProvider := shipInfra.NewLogShippingProvider("default-carrier")
        shippingService := shipApp.NewService(shipmentRepo, shippingProvider, shippingOutboxPublisher)
        shippingHandler := shipTransport.NewHandler(shippingService)

        // --- Wire up reviews ---
        reviewRepo := revInfra.NewReviewRepository(db)
        vpChecker := revInfra.NewOrderVerifiedPurchaseChecker(db)
        reviewService := revApp.NewService(reviewRepo, vpChecker)
        reviewService.SetAuthorNames(revInfra.NewUserNameResolver(db))
        reviewService.RequireVerifiedPurchase(true) // only customers who received the product can review it
        reviewHandler := revTransport.NewHandler(reviewService)

        // --- Wire up wishlist ---
        wishlistRepo := wishInfra.NewWishlistRepository(db)
        productValidator := wishInfra.NewProductRepositoryValidator(productRepo)
        wishlistService := wishApp.NewService(wishlistRepo, productValidator, nil)
        wishlistHandler := wishTransport.NewHandler(wishlistService)

        // --- Wire up inventory ---
        stockRepo := invInfra.NewStockRepository(db)
        reservationRepo := invInfra.NewReservationRepository(db)
        movementRepo := invInfra.NewMovementRepository(db)
        warehouseRepo := invInfra.NewWarehouseRepository(db)
        inventoryService := invApp.NewService(stockRepo, reservationRepo, movementRepo, warehouseRepo, inventoryOutboxPublisher)
        inventoryService.SetAuditor(auditor)
        // Wire business metrics: inventory reservation failures
        if allMetrics != nil {
                inventoryService.SetBusinessMetrics(allMetrics.Business)
        }
        inventoryHandler := invTransport.NewHandler(inventoryService)

        // --- Wire up vendors ---
        vendorRepo := vendInfra.NewRepository(db)
        vendorService := vendApp.NewService(vendorRepo, userRepo)
        vendorHandler := vendTransport.NewHandler(vendorService)
        vendorMW := vendTransport.VendorMiddleware(vendorService)

        // --- Wire up orders ---
        orderRepo := ordInfra.NewOrderRepository(db)
        orderService := ordApp.NewService(orderRepo, orderOutboxPublisher)
        orderService.SetAuditor(auditor)
        // Wire business metrics: orders created
        if allMetrics != nil {
                orderService.SetBusinessMetrics(allMetrics.Business)
        }
        orderService.SetStock(ordInfra.NewStockReserver(inventoryService))
        couponApplier := ordInfra.NewCouponApplier(db)
        orderService.SetCoupons(couponApplier)
        // --- Wire up purchases (stock receiving) ---
        purchaseService := purApp.NewService(
                purInfra.NewRepository(db),
                purInfra.NewStockReceiver(inventoryService),
                purInfra.NewCostCatalog(productRepo),
        )

        orderHandler := ordTransport.NewHandler(orderService)
        // Server-side pricing: order line name/sku/price/vendor come from the catalog.
        checkoutCfg := ordTransport.DefaultCheckoutConfig()
        for _, w := range []ordTransport.WalletConfig{
                {ID: "bkash", Label: "bKash", Number: os.Getenv("PAYMENT_BKASH_NUMBER"), Type: "Personal"},
                {ID: "nagad", Label: "Nagad", Number: os.Getenv("PAYMENT_NAGAD_NUMBER"), Type: "Personal"},
                {ID: "rocket", Label: "Rocket", Number: os.Getenv("PAYMENT_ROCKET_NUMBER"), Type: "Personal"},
        } {
                if w.Number != "" { // only wallets you configured are offered to customers
                        checkoutCfg.Wallets = append(checkoutCfg.Wallets, w)
                }
        }
        orderHandler.SetCheckoutConfig(checkoutCfg)
        // Delivery charges are set by the shop in the admin panel (Shipping -> Delivery charges).
        settingsService := setApp.NewService(setInfra.NewRepository(db))
        orderHandler.SetDelivery(settingsService)
        orderHandler.SetCoupons(couponApplier)
        orderHandler.SetCatalog(ordInfra.NewCatalogResolver(productRepo, vendorRepo))

        // --- Wire up payment webhooks ---
        paymentRepo := payInfra.NewPaymentRepository(db)
        // Webhook secrets must be loaded from environment variables — never hardcode.
        webhookSecrets := loadWebhookSecrets()
        if len(webhookSecrets) == 0 {
                log.Warnw("no webhook secrets configured via WEBHOOK_SECRET_<PROVIDER> env vars")
        }
        webhookVerifier := payInfra.NewFakeSignatureVerifier(webhookSecrets)
        webhookEventStore := payInfra.NewWebhookEventStore(db)
        webhookService := payApp.NewWebhookService(paymentRepo, webhookVerifier, webhookEventStore, paymentOutboxPublisher)
        webhookHandler := payTransport.NewWebhookHandler(webhookService)

        // Wire business metrics: payment success/failure
        // The payment application service is used indirectly through webhook service.
        // Business metrics are wired via the payment service if one is created directly.

        // Health checkers
        healthHandler := health.NewHandler()
        healthHandler.Register(&mongoHealthChecker{client: mongoClient})
        healthHandler.Register(&natsHealthChecker{client: natsClient})
        healthHandler.Register(&natsJetStreamChecker{client: natsClient})
        healthHandler.Register(&kvHealthChecker{client: natsClient, buckets: requiredBuckets})

        // Create Fiber app
        app := fiber.New(fiber.Config{
                ReadTimeout:  cfg.Server.ReadTimeout,
                WriteTimeout: cfg.Server.WriteTimeout,
                IdleTimeout:  cfg.Server.IdleTimeout,
                BodyLimit:    1 * 1024 * 1024, // 1 MB max request body
                ErrorHandler: defaultErrorHandler,
                AppName:      "ecommerce-api",
        })

        // Global middleware
        app.Use(recover.New())
        app.Use(middleware.RequestID())
        if allMetrics != nil {
                app.Use(middleware.Telemetry(middleware.TelemetryConfig{
                        HTTPMetrics: allMetrics.HTTP,
                }))
        }
        app.Use(middleware.SecurityHeaders())
        app.Use(cors.New(cors.Config{
                AllowOrigins: strings.Join(cfg.Security.CORSAllowedOrigins, ","),
                AllowMethods: strings.Join(cfg.Security.CORSAllowedMethods, ","),
                AllowHeaders: strings.Join(cfg.Security.CORSAllowedHeaders, ","),
        }))

        // Health routes
        app.Get("/health", healthHandler.Health)
        app.Get("/health/live", healthHandler.Live)
        app.Get("/health/ready", healthHandler.Ready)

        // Auth routes
        authTransport.RegisterAuthRoutes(app, authHandler)
        userTransport.RegisterUserRoutes(app, userHandler, authMW)

        // Product & Category routes
        prodTransport.RegisterProductRoutes(app, productHandler, authMW, adminMW, vendorMW)
        vendTransport.RegisterVendorRoutes(app, vendorHandler, authMW, adminMW, vendorMW)
        catTransport.RegisterCategoryRoutes(app, categoryHandler, authMW, adminMW)
        invTransport.RegisterInventoryRoutes(app, inventoryHandler, authMW, adminMW)
        ordTransport.RegisterOrderRoutes(app, orderHandler, authMW, adminMW)
        purTransport.RegisterRoutes(app, purTransport.NewHandler(purchaseService), authMW, adminMW)
        repTransport.RegisterRoutes(app, repTransport.NewHandler(repInfra.NewRepository(db)), authMW, adminMW)

        // --- Wire up support messaging ---
        supportService := supApp.NewService(supInfra.NewRepository(db), supInfra.NewUserLookup(db))
        supportService.SetOrderChecker(supInfra.NewOrderChecker(db))
        supportService.SetNotifier(supInfra.NewEmailNotifier(emailSender, brand, os.Getenv("SUPPORT_EMAIL")))
        staffMW := authTransport.RequireRole(userDomain.RoleSuperAdmin, userDomain.RoleAdmin, userDomain.RoleManager, userDomain.RoleSupport)
        supTransport.RegisterRoutes(app, supTransport.NewHandler(supportService), authMW, staffMW)
        setTransport.RegisterRoutes(app, setTransport.NewHandler(settingsService), authMW, adminMW)
        payTransport.RegisterWebhookRoutes(app, webhookHandler)
        shipTransport.RegisterShippingRoutes(app, shippingHandler, authMW, adminMW)
        revTransport.RegisterReviewRoutes(app, reviewHandler, authMW)
        wishTransport.RegisterWishlistRoutes(app, wishlistHandler, authMW)

        // --- Wire up marketing (admin CRUD) ---
        mktCampaignRepo := mktInfra.NewCampaignRepository(db)
        mktSegmentRepo := mktInfra.NewSegmentRepository(db)
        mktPromotionRepo := mktInfra.NewPromotionRepository(db)
        mktService := mktApp.NewService(mktCampaignRepo, mktSegmentRepo, mktPromotionRepo)
        mktService.SetAuditor(auditor)
        mktHandler := mktTransport.NewHandler(mktService)
        mktTransport.RegisterMarketingRoutes(app, mktHandler, authMW, adminMW)

        // --- Wire up analytics (admin read + JetStream worker) ---
        analyticsRepo := analyticsInfra.NewAnalyticsRepository(db)
        analyticsService := analyticsApp.NewService(analyticsRepo)
        analyticsHandler := analyticsTransport.NewHandler(analyticsService)
        analyticsTransport.RegisterAnalyticsRoutes(app, analyticsHandler, authMW, adminMW)

        // Start analytics JetStream worker (non-blocking)
        analyticsProcessor := analyticsWorker.NewProcessor(analyticsWorker.ProcessorDependencies{
                Repo: analyticsRepo,
                Log:  log,
        })
        analyticsSubscriber := eventstore.NewSubscriber(natsClient.JS(), log, analyticsProcessor.Handle())
        // Wire NATS consumer metrics for the analytics subscriber.
        if allMetrics != nil {
                analyticsSubscriber.SetNATSMetrics(allMetrics.NATS)
        }
        go func() {
                if err := analyticsSubscriber.Subscribe(ctx, eventstore.SubscriberOptions{
                        Stream:        eventstore.StreamAnalytics,
                        ConsumerName:  "analytics-worker",
                        Description:   "Persists analytics events to analytics storage",
                        MaxDeliveries: 10,
                }); err != nil {
                        log.Errorw("analytics worker stopped", "error", err)
                }
        }()
        log.Info("analytics worker started")

        // --- Wire up admin platform ---
        adminService := adminApp.NewService(adminApp.Dependencies{
                UserRepo:      userRepo,
                CustomerRepo:  customerRepo,
                ReviewRepo:    reviewRepo,
                AuditListRepo: auditRepo,
                Auditor:       auditor,
        })
        adminHandler := adminTransport.NewHandler(adminService)
        adminTransport.RegisterAdminRoutes(app, adminHandler, authMW, adminMW)

        // Start server
        go func() {
                addr := cfg.ServerAddr()
                log.Infow("http server listening", "addr", addr)
                if err := app.Listen(addr); err != nil {
                        log.Fatalw("server error", "error", err)
                }
        }()

        // Graceful shutdown
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        sig := <-quit
        log.Infow("shutting down", "signal", sig.String())

        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
        defer shutdownCancel()

        if err := app.ShutdownWithContext(shutdownCtx); err != nil {
                log.Errorw("server shutdown error", "error", err)
        }

        log.Info("server stopped")
}

func defaultErrorHandler(c *fiber.Ctx, err error) error {
        // Errors Fiber raises itself (unknown route, wrong method, body too large...) keep their status.
        var fe *fiber.Error
        if stderrors.As(err, &fe) {
                code := "BAD_REQUEST"
                switch fe.Code {
                case http.StatusNotFound:
                        code = "NOT_FOUND"
                case http.StatusMethodNotAllowed:
                        code = "METHOD_NOT_ALLOWED"
                case http.StatusRequestEntityTooLarge:
                        code = "PAYLOAD_TOO_LARGE"
                }
                return c.Status(fe.Code).JSON(fiber.Map{
                        "success": false,
                        "error":   fiber.Map{"code": code, "message": fe.Message},
                })
        }
        return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
                "success": false,
                "error": fiber.Map{
                        "code":    "INTERNAL_ERROR",
                        "message": "An unexpected error occurred",
                },
        })
}

// --- Health checkers ---

type mongoHealthChecker struct {
        client *mongodb.Client
}

func (m *mongoHealthChecker) Name() string { return "mongodb" }
func (m *mongoHealthChecker) Check(ctx context.Context) error {
        return m.client.HealthCheck(ctx)
}

type natsHealthChecker struct {
        client *nats.Client
}

func (n *natsHealthChecker) Name() string { return "nats" }
func (n *natsHealthChecker) Check(ctx context.Context) error {
        return n.client.HealthCheck(ctx)
}

type natsJetStreamChecker struct {
        client *nats.Client
}

func (n *natsJetStreamChecker) Name() string { return "jetstream" }
func (n *natsJetStreamChecker) Check(ctx context.Context) error {
        return n.client.JetStreamHealthCheck(ctx)
}

type kvHealthChecker struct {
        client  *nats.Client
        buckets []string
}

func (k *kvHealthChecker) Name() string { return "nats_kv" }
func (k *kvHealthChecker) Check(ctx context.Context) error {
        return k.client.KVHealthCheck(ctx, k.buckets)
}

// loadWebhookSecrets reads webhook HMAC secrets from environment variables.
// Expected format: WEBHOOK_SECRET_<PROVIDER>=<secret>, e.g. WEBHOOK_SECRET_FAKE=abc123.
func loadWebhookSecrets() map[string]string {
        secrets := make(map[string]string)
        for _, env := range os.Environ() {
                if !strings.HasPrefix(env, "WEBHOOK_SECRET_") {
                        continue
                }
                parts := strings.SplitN(env, "=", 2)
                if len(parts) != 2 || parts[1] == "" {
                        continue
                }
                provider := strings.TrimPrefix(parts[0], "WEBHOOK_SECRET_")
                secrets[strings.ToLower(provider)] = parts[1]
        }
        return secrets
}

// envOr returns the environment variable, or the fallback when it is empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

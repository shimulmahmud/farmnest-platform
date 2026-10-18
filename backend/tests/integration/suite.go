// Package integration provides Testcontainers-based integration tests
// for the ecommerce backend. Tests use real MongoDB and NATS instances
// running in Docker containers, with a full Fiber HTTP server.
package integration

import (
        "context"
        "encoding/json"
        "fmt"
        "time"

        "go.mongodb.org/mongo-driver/v2/bson"
        "go.mongodb.org/mongo-driver/v2/mongo"

        adminApp "github.com/ecommerce/backend/internal/admin/application"
        adminTransport "github.com/ecommerce/backend/internal/admin/transport"
        "github.com/ecommerce/backend/internal/analytics/infrastructure"
        analyticsApp "github.com/ecommerce/backend/internal/analytics/application"
        analyticsTransport "github.com/ecommerce/backend/internal/analytics/transport"
        authApp "github.com/ecommerce/backend/internal/auth/application"
        authInfra "github.com/ecommerce/backend/internal/auth/infrastructure"
        authTransport "github.com/ecommerce/backend/internal/auth/transport"
        catApp "github.com/ecommerce/backend/internal/categories/application"
        catInfra "github.com/ecommerce/backend/internal/categories/infrastructure"
        catTransport "github.com/ecommerce/backend/internal/categories/transport"
        "github.com/ecommerce/backend/internal/customers/infrastructure"
        invApp "github.com/ecommerce/backend/internal/inventory/application"
        invInfra "github.com/ecommerce/backend/internal/inventory/infrastructure"
        invTransport "github.com/ecommerce/backend/internal/inventory/transport"
        mktApp "github.com/ecommerce/backend/internal/marketing/application"
        mktInfra "github.com/ecommerce/backend/internal/marketing/infrastructure"
        mktTransport "github.com/ecommerce/backend/internal/marketing/transport"
        "github.com/ecommerce/backend/internal/middleware"
        ordApp "github.com/ecommerce/backend/internal/orders/application"
        ordDomain "github.com/ecommerce/backend/internal/orders/domain"
        ordInfra "github.com/ecommerce/backend/internal/orders/infrastructure"
        ordTransport "github.com/ecommerce/backend/internal/orders/transport"
        payApp "github.com/ecommerce/backend/internal/payments/application"
        payDomain "github.com/ecommerce/backend/internal/payments/domain"
        payInfra "github.com/ecommerce/backend/internal/payments/infrastructure"
        payTransport "github.com/ecommerce/backend/internal/payments/transport"
        revApp "github.com/ecommerce/backend/internal/reviews/application"
        revInfra "github.com/ecommerce/backend/internal/reviews/infrastructure"
        revTransport "github.com/ecommerce/backend/internal/reviews/transport"
        shipApp "github.com/ecommerce/backend/internal/shipping/application"
        shipInfra "github.com/ecommerce/backend/internal/shipping/infrastructure"
        shipTransport "github.com/ecommerce/backend/internal/shipping/transport"
        prodApp "github.com/ecommerce/backend/internal/products/application"
        prodInfra "github.com/ecommerce/backend/internal/products/infrastructure"
        prodTransport "github.com/ecommerce/backend/internal/products/transport"
        userDomain "github.com/ecommerce/backend/internal/users/domain"
        userInfra "github.com/ecommerce/backend/internal/users/infrastructure"
        userTransport "github.com/ecommerce/backend/internal/users/transport"
        wlApp "github.com/ecommerce/backend/internal/wishlist/application"
        wlInfra "github.com/ecommerce/backend/internal/wishlist/infrastructure"
        wlTransport "github.com/ecommerce/backend/internal/wishlist/transport"
        "github.com/ecommerce/backend/pkg/audit"
        "github.com/ecommerce/backend/pkg/eventstore"
        "github.com/ecommerce/backend/pkg/logger"
        "github.com/ecommerce/backend/pkg/mongodb"
        "github.com/ecommerce/backend/pkg/nats"
        "github.com/ecommerce/backend/pkg/outbox"

        "github.com/gofiber/fiber/v2"
        "github.com/gofiber/fiber/v2/middleware/cors"
        "github.com/gofiber/fiber/v2/middleware/recover"
        "github.com/stretchr/testify/suite"
        "github.com/testcontainers/testcontainers-go"
        "github.com/testcontainers/testcontainers-go/modules/mongodb"
        "github.com/testcontainers/testcontainers-go/modules/nats"
)

const (
        testJWTSecret     = "integration-test-secret-key-32chars!!"
        testJWTIssuer     = "ecommerce-test"
        testDBName        = "ecommerce_test"
        testWebhookSecret = "whsec-integration-test-hmac-key"
)

// IntegrationSuite provides Testcontainers-managed MongoDB and NATS,
// a fully-wired Fiber app, and HTTP test helpers.
type IntegrationSuite struct {
        suite.Suite

        mongoContainer testcontainers.Container
        natsContainer  testcontainers.Container

        mongoClient *mongodb.Client
        natsClient  *nats.Client

        app    *fiber.App
        appURL string

        // Direct repository access for MongoDB state assertions.
        userRepo    userDomain.UserRepository
        orderRepo   ordDomain.OrderRepository
        paymentRepo payDomain.PaymentRepository

        // Direct service access for concurrency tests.
        inventoryService *invApp.Service
        marketingService *mktApp.Service

        // Webhook verifier for signing test payloads.
        webhookVerifier *payInfra.FakeSignatureVerifier
}

// SetupSuite starts containers and wires the full application.
func (s *IntegrationSuite) SetupSuite() {
        ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
        defer cancel()

        // --- MongoDB ---
        mongoContainer, err := mongodb.RunContainer(ctx, mongodb.NewRequest(ctx, "mongo:7"))
        s.Require().NoError(err, "failed to start MongoDB container")
        s.mongoContainer = mongoContainer

        mongoURI, err := mongoContainer.ConnectionString(ctx)
        s.Require().NoError(err)

        mongoClient, err := mongodb.New(ctx, mongoURI, testDBName, 10, 2, 15*time.Second)
        s.Require().NoError(err, "failed to connect to MongoDB")
        s.mongoClient = mongoClient

        // --- NATS ---
        natsContainer, err := nats.RunContainer(ctx, nats.NewRequest(ctx, "nats:2.10-alpine"))
        s.Require().NoError(err, "failed to start NATS container")
        s.natsContainer = natsContainer

        natsURL, err := natsContainer.ConnectionString(ctx)
        s.Require().NoError(err)

        natsClient, err := nats.New(ctx, natsURL, "test-cluster", 1*time.Second, 3, 5*time.Second)
        s.Require().NoError(err, "failed to connect to NATS")
        s.natsClient = natsClient

        // --- Indexes ---
        db := mongoClient.Database()
        indexFns := map[string]func(context.Context, *mongo.Database) error{
                "users":      userInfra.EnsureIndexes,
                "tokens":     authInfra.EnsureTokenIndexes,
                "customers":  customerInfra.EnsureIndexes,
                "products":   prodInfra.EnsureIndexes,
                "categories": catInfra.EnsureIndexes,
                "inventory":  invInfra.EnsureIndexes,
                "orders":     ordInfra.EnsureIndexes,
                "payments":   payInfra.EnsureIndexes,
                "shipments":  shipInfra.EnsureIndexes,
                "reviews":    revInfra.EnsureIndexes,
                "wishlists":  wlInfra.EnsureIndexes,
                "marketing":  mktInfra.EnsureIndexes,
                "outbox":     outbox.EnsureIndexes,
                "audit_logs": audit.EnsureIndexes,
                "webhooks":   payInfra.EnsureWebhookIndexes,
        }
        for name, fn := range indexFns {
                s.Require().NoError(fn(ctx, db), "failed to create %s indexes", name)
        }

        // --- JetStream streams ---
        streamMgr := eventstore.NewStreamManager(natsClient.JS())
        s.Require().NoError(streamMgr.EnsureStreams(ctx, eventstore.DefaultStreamConfigs()))

        // --- KV buckets ---
        for _, bucket := range []struct {
                name    string
                history uint8
        }{
                {"PRODUCT_CACHE", 1}, {"CATEGORY_CACHE", 1}, {"SESSION_STATE", 0},
                {"IDEMPOTENCY", 0}, {"RATE_LIMIT", 0}, {"CHECKOUT_STATE", 0}, {"DISTRIBUTED_LOCKS", 0},
        } {
                _, err := natsClient.InitKVBucket(ctx, bucket.name, bucket.history)
                s.Require().NoError(err, "failed to init KV bucket %s", bucket.name)
        }

        // --- Wire services ---
        log, _ := logger.New("warn", "json")
        jwtService := authInfra.NewJWTService(testJWTSecret, 15*time.Minute, 168*time.Hour, testJWTIssuer)
        passwordHasher := authInfra.NewPasswordHasher(4)

        userRepo := userInfra.NewUserRepository(db)
        tokenRepo := authInfra.NewTokenRepository(db)
        customerRepo := customerInfra.NewCustomerRepository(db)
        s.userRepo = userRepo

        authService := authApp.NewService(userRepo, tokenRepo, customerRepo, jwtService, passwordHasher, authApp.DefaultAuthConfig(), 168*time.Hour)
        authHandler := authTransport.NewHandler(authService)
        userHandler := userTransport.NewHandler(userRepo)
        authMW := authTransport.AuthMiddleware(authService)
        adminMW := authTransport.RequireRole(userDomain.RoleSuperAdmin, userDomain.RoleAdmin)

        // Audit
        auditRepo := audit.NewRepository(db)
        auditor := audit.NewAuditor(auditRepo, log)

        // Outbox
        outboxRepo := outbox.NewOutboxRepository(db)
        eventPublisher := eventstore.NewPublisher(natsClient.JS(), log)
        outboxWorker := outbox.NewWorker(outboxRepo, eventPublisher, log, outbox.DefaultWorkerConfig())
        outboxWorker.Start(ctx)

        // --- Domain wiring ---
        productKV, _ := natsClient.KV("PRODUCT_CACHE")
        categoryKV, _ := natsClient.KV("CATEGORY_CACHE")

        productRepo := prodInfra.NewProductRepository(db)
        productService := prodApp.NewService(productRepo, prodInfra.NewProductCache(productKV))
        productService.SetAuditor(auditor)
        productService.SetStockChecker(prodInfra.NewStockChecker(db))
        productService.SetEvents(outbox.NewProductPublisherAdapter(outboxRepo))
        productHandler := prodTransport.NewHandler(productService)

        categoryRepo := catInfra.NewCategoryRepository(db)
        categoryService := catApp.NewService(categoryRepo, catInfra.NewCategoryCache(categoryKV))
        categoryHandler := catTransport.NewHandler(categoryService)

        stockRepo := invInfra.NewStockRepository(db)
        reservationRepo := invInfra.NewReservationRepository(db)
        movementRepo := invInfra.NewMovementRepository(db)
        warehouseRepo := invInfra.NewWarehouseRepository(db)
        inventoryService := invApp.NewService(stockRepo, reservationRepo, movementRepo, warehouseRepo, outbox.NewInventoryPublisherAdapter(outboxRepo))
        inventoryHandler := invTransport.NewHandler(inventoryService)
        s.inventoryService = inventoryService

        orderRepo := ordInfra.NewOrderRepository(db)
        orderService := ordApp.NewService(orderRepo, outbox.NewOrderPublisherAdapter(outboxRepo))
        orderService.SetAuditor(auditor)
        orderHandler := ordTransport.NewHandler(orderService)
        s.orderRepo = orderRepo

        paymentRepo := payInfra.NewPaymentRepository(db)
        webhookSecrets := map[string]string{"fake": testWebhookSecret}
        webhookVerifier := payInfra.NewFakeSignatureVerifier(webhookSecrets)
        webhookEventStore := payInfra.NewWebhookEventStore(db)
        webhookService := payApp.NewWebhookService(paymentRepo, webhookVerifier, webhookEventStore, outbox.NewPaymentPublisherAdapter(outboxRepo))
        webhookHandler := payTransport.NewWebhookHandler(webhookService)
        s.paymentRepo = paymentRepo
        s.webhookVerifier = webhookVerifier

        shipmentRepo := shipInfra.NewShipmentRepository(db)
        shippingService := shipApp.NewService(shipmentRepo, shipInfra.NewLogShippingProvider("test-carrier"), outbox.NewShippingPublisherAdapter(outboxRepo))
        shippingHandler := shipTransport.NewHandler(shippingService)

        // Reviews
        reviewRepo := revInfra.NewReviewRepository(db)
        reviewService := revApp.NewService(reviewRepo, nil)
        reviewHandler := revTransport.NewHandler(reviewService)

        adminService := adminApp.NewService(adminApp.Dependencies{
                UserRepo:      userRepo,
                CustomerRepo:  customerRepo,
                ReviewRepo:    reviewRepo,
                AuditListRepo: auditRepo,
                Auditor:       auditor,
        })
        adminHandler := adminTransport.NewHandler(adminService)

        analyticsRepoInst := analyticsInfra.NewAnalyticsRepository(db)
        analyticsService := analyticsApp.NewService(analyticsRepoInst)
        analyticsHandler := analyticsTransport.NewHandler(analyticsService)

        // Wishlist
        wishlistRepo := wlInfra.NewWishlistRepository(db)
        wishlistValidator := wlInfra.NewProductRepositoryValidator(productRepo)
        wishlistService := wlApp.NewService(wishlistRepo, wishlistValidator, nil)
        wishlistHandler := wlTransport.NewHandler(wishlistService)

        // Marketing
        mktCampaignRepo := mktInfra.NewCampaignRepository(db)
        mktSegmentRepo := mktInfra.NewSegmentRepository(db)
        mktPromotionRepo := mktInfra.NewPromotionRepository(db)
        mktService := mktApp.NewService(mktCampaignRepo, mktSegmentRepo, mktPromotionRepo)
        mktService.SetAuditor(auditor)
        s.marketingService = mktService
        mktHandler := mktTransport.NewHandler(mktService)

        // --- Fiber app ---
        app := fiber.New(fiber.Config{
                BodyLimit:    1 * 1024 * 1024,
                ErrorHandler: defaultTestErrorHandler,
                AppName:      "ecommerce-integration-test",
        })

        app.Use(recover.New())
        app.Use(middleware.RequestID())
        app.Use(middleware.SecurityHeaders())
        app.Use(cors.New(cors.Config{
                AllowOrigins: "http://localhost:3000",
                AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
                AllowHeaders: "Content-Type,Authorization,X-Request-ID",
        }))

        app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

        authTransport.RegisterAuthRoutes(app, authHandler)
        userTransport.RegisterUserRoutes(app, userHandler, authMW)
        prodTransport.RegisterProductRoutes(app, productHandler, authMW, adminMW, authTransport.RequireRole(userDomain.RoleVendor))
        catTransport.RegisterCategoryRoutes(app, categoryHandler, authMW, adminMW)
        invTransport.RegisterInventoryRoutes(app, inventoryHandler, authMW, adminMW)
        ordTransport.RegisterOrderRoutes(app, orderHandler, authMW, adminMW)
        payTransport.RegisterWebhookRoutes(app, webhookHandler)
        shipTransport.RegisterShippingRoutes(app, shippingHandler, authMW, adminMW)
        adminTransport.RegisterAdminRoutes(app, adminHandler, authMW, adminMW)
        analyticsTransport.RegisterAnalyticsRoutes(app, analyticsHandler, authMW, adminMW)
        revTransport.RegisterReviewRoutes(app, reviewHandler, authMW)
        wlTransport.RegisterWishlistRoutes(app, wishlistHandler, authMW)
        mktTransport.RegisterMarketingRoutes(app, mktHandler, authMW, adminMW)

        s.app = app
        s.appURL = "http://localhost"
}

// TearDownSuite stops containers.
func (s *IntegrationSuite) TearDownSuite() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if s.mongoClient != nil {
                s.mongoClient.Close(ctx)
        }
        if s.natsClient != nil {
                s.natsClient.Close()
        }
        if s.mongoContainer != nil {
                s.mongoContainer.Terminate(ctx)
        }
        if s.natsContainer != nil {
                s.natsContainer.Terminate(ctx)
        }
}

// SetupTest drops all collections for test isolation.
func (s *IntegrationSuite) SetupTest() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        db := s.mongoClient.Database()
        names, err := db.ListCollectionNames(ctx, bson.M{})
        s.Require().NoError(err)
        for _, name := range names {
                s.Require().NoError(db.Collection(name).Drop(ctx), "drop %s", name)
        }
}

// ==================== HTTP Helpers ====================

// JSONResponse is a parsed HTTP response.
type JSONResponse struct {
        StatusCode int
        Body       map[string]interface{}
        Headers    map[string]string
}

// GET sends an authenticated GET request.
func (s *IntegrationSuite) GET(path, token string) *JSONResponse {
        return s.doRequest("GET", path, nil, token)
}

// POST sends a POST request with JSON body.
func (s *IntegrationSuite) POST(path string, body interface{}, token string) *JSONResponse {
        return s.doRequest("POST", path, body, token)
}

// PATCH sends a PATCH request with JSON body.
func (s *IntegrationSuite) PATCH(path string, body interface{}, token string) *JSONResponse {
        return s.doRequest("PATCH", path, body, token)
}

// POSTWebhook sends a raw POST (for webhook payloads).
func (s *IntegrationSuite) POSTWebhook(path string, body []byte, headers map[string]string) *JSONResponse {
        req := s.app.Post(path)
        for k, v := range headers {
                req.Set(k, v)
        }
        req.Set("Content-Type", "application/json")
        req.Body(body)
        resp, err := req.Bytes()
        s.Require().NoError(err)
        return parseFiberResponse(resp)
}

// DELETE sends a DELETE request.
func (s *IntegrationSuite) DELETE(path, token string) *JSONResponse {
        req := s.app.Delete(path)
        if token != "" {
                req.Set("Authorization", "Bearer "+token)
        }
        resp, err := req.Bytes()
        s.Require().NoError(err)
        return parseFiberResponse(resp)
}

// PUT sends a PUT request with JSON body.
func (s *IntegrationSuite) PUT(path string, body interface{}, token string) *JSONResponse {
        req := s.app.Put(path)
        if token != "" {
                req.Set("Authorization", "Bearer "+token)
        }
        req.Set("Content-Type", "application/json")
        req.JSON(body)
        resp, err := req.Bytes()
        s.Require().NoError(err)
        return parseFiberResponse(resp)
}

func (s *IntegrationSuite) doRequest(method, path string, body interface{}, token string) *JSONResponse {
        var resp *fiber.Response
        var err error

        switch method {
        case "GET":
                req := s.app.Get(path)
                if token != "" {
                        req.Set("Authorization", "Bearer "+token)
                }
                resp, err = req.Bytes()
        case "POST":
                req := s.app.Post(path)
                if token != "" {
                        req.Set("Authorization", "Bearer "+token)
                }
                req.Set("Content-Type", "application/json")
                req.JSON(body)
                resp, err = req.Bytes()
        case "PATCH":
                req := s.app.Patch(path)
                if token != "" {
                        req.Set("Authorization", "Bearer "+token)
                }
                req.Set("Content-Type", "application/json")
                req.JSON(body)
                resp, err = req.Bytes()
        case "PUT":
                req := s.app.Put(path)
                if token != "" {
                        req.Set("Authorization", "Bearer "+token)
                }
                req.Set("Content-Type", "application/json")
                req.JSON(body)
                resp, err = req.Bytes()
        case "DELETE":
                req := s.app.Delete(path)
                if token != "" {
                        req.Set("Authorization", "Bearer "+token)
                }
                resp, err = req.Bytes()
        default:
                s.FailNow(fmt.Sprintf("unsupported method: %s", method))
                return nil
        }
        s.Require().NoError(err)
        return parseFiberResponse(resp)
}

// ==================== Auth Helpers ====================

// AuthResult holds tokens from registration or login.
type AuthResult struct {
        UserID       string
        AccessToken  string
        RefreshToken string
        Email        string
}

// RegisterUser creates a new user and returns auth tokens.
func (s *IntegrationSuite) RegisterUser(email, password, firstName, lastName string) *AuthResult {
        resp := s.POST("/api/v1/auth/register", map[string]string{
                "email":      email,
                "password":   password,
                "first_name": firstName,
                "last_name":  lastName,
        }, "")
        s.Require().Equal(201, resp.StatusCode, "registration failed: %v", resp.Body)

        data := resp.Body["data"].(map[string]interface{})
        tokens := data["tokens"].(map[string]interface{})
        user := data["user"].(map[string]interface{})

        return &AuthResult{
                UserID:       user["id"].(string),
                AccessToken:  tokens["access_token"].(string),
                RefreshToken: tokens["refresh_token"].(string),
                Email:        email,
        }
}

// LoginUser logs in an existing user.
func (s *IntegrationSuite) LoginUser(email, password string) *AuthResult {
        resp := s.POST("/api/v1/auth/login", map[string]string{
                "email":    email,
                "password": password,
        }, "")
        s.Require().Equal(200, resp.StatusCode, "login failed: %v", resp.Body)

        data := resp.Body["data"].(map[string]interface{})
        tokens := data["tokens"].(map[string]interface{})
        user := data["user"].(map[string]interface{})

        return &AuthResult{
                UserID:       user["id"].(string),
                AccessToken:  tokens["access_token"].(string),
                RefreshToken: tokens["refresh_token"].(string),
                Email:        email,
        }
}

// ==================== Assertion Helpers ====================

// assertSuccess checks that the response was successful.
func (s *IntegrationSuite) assertSuccess(resp *JSONResponse) {
        s.True(resp.Body["success"].(bool), "expected success, got: %v", resp.Body)
}

// assertErrorCode checks that the response is an error with the given code.
func (s *IntegrationSuite) assertErrorCode(resp *JSONResponse, code string) {
        s.False(resp.Body["success"].(bool), "expected error, got success")
        errBody := resp.Body["error"].(map[string]interface{})
        s.Equal(code, errBody["code"], "error code mismatch")
}

// dataField returns the data field from a successful response.
func (s *IntegrationSuite) dataField(resp *JSONResponse) map[string]interface{} {
        s.assertSuccess(resp)
        return resp.Body["data"].(map[string]interface{})
}

// WaitEventually polls until fn returns true or timeout.
func (s *IntegrationSuite) WaitEventually(timeout time.Duration, fn func() bool, msg string) {
        deadline := time.Now().Add(timeout)
        for time.Now().Before(deadline) {
                if fn() {
                        return
                }
                time.Sleep(100 * time.Millisecond)
        }
        s.Fail(msg)
}

// ==================== Internal ====================

func parseFiberResponse(resp *fiber.Response) *JSONResponse {
        var body map[string]interface{}
        _ = json.Unmarshal(resp.Body(), &body)

        headers := make(map[string]string)
        resp.Header.VisitAll(func(k, v []byte) {
                headers[string(k)] = string(v)
        })

        return &JSONResponse{StatusCode: resp.StatusCode(), Body: body, Headers: headers}
}

func defaultTestErrorHandler(c *fiber.Ctx, err error) error {
        return c.Status(500).JSON(fiber.Map{
                "success": false,
                "error": fiber.Map{
                        "code":    "INTERNAL_ERROR",
                        "message": "An unexpected error occurred",
                },
        })
}
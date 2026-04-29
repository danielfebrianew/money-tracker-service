package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	"money-management-service/internal/database"
	"money-management-service/internal/handler"
	appmw "money-management-service/internal/middleware"
	adminmodule "money-management-service/internal/modules/admin"
	authmodule "money-management-service/internal/modules/auth"
	balancemodule "money-management-service/internal/modules/balance"
	dashboardmodule "money-management-service/internal/modules/dashboard"
	groupsmodule "money-management-service/internal/modules/groups"
	paymentsmodule "money-management-service/internal/modules/payments"
	tokensmodule "money-management-service/internal/modules/tokens"
	transactions "money-management-service/internal/modules/transactions"
	usersmodule "money-management-service/internal/modules/users"
	webhookmodule "money-management-service/internal/modules/webhook"
	"money-management-service/internal/repository"
	"money-management-service/internal/service"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := database.ConnectPostgres(ctx, cfg)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	migrationPath := migrationPath()
	if err := database.RunMigrations(cfg, migrationPath); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	redisClient, err := database.ConnectRedis(ctx, cfg)
	if err != nil {
		log.Printf("redis disabled: %v", err)
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	store := repository.NewStore(db)
	appCache := cache.New(redisClient)

	authModule := authmodule.NewModule(cfg, store, appCache)
	authService := authModule.Service
	userModule := usersmodule.NewModule(store, appCache)
	balanceModule := balancemodule.NewModule(store)
	balanceService := balanceModule.Service
	parser := service.NewSmartParser(cfg, appCache)
	fonnte := service.NewFonnteClient(cfg)
	paymentModule := paymentsmodule.NewModule(store, appCache)
	tokenModule := tokensmodule.NewModule(store)
	transactionModule := transactions.NewModule(store, appCache, parser)
	transactionService := transactionModule.Service
	dashboardModule := dashboardmodule.NewModule(appCache, store)
	groupModule := groupsmodule.NewModule(store, appCache, transactionService)
	referralService := service.NewReferralService(cfg, store)
	adminService := service.NewAdminService(store, appCache)
	adminModule := adminmodule.NewModule(authService, adminService, paymentModule.Service, adminmodule.NewRepository(store))
	webhookService := service.NewWebhookService(cfg, store, appCache, parser, fonnte, transactionService)
	webhookModule := webhookmodule.NewModule(webhookService, webhookmodule.NewRepository(store))

	if err := authService.SeedAdmin(ctx); err != nil {
		log.Fatalf("seed admin: %v", err)
	}

	h := handler.New(handler.Dependencies{
		Auth:         authModule,
		User:         userModule,
		Balance:      balanceModule,
		Tokens:       tokenModule,
		Payments:     paymentModule,
		Transactions: transactionModule,
		Dashboard:    dashboardModule,
		Groups:       groupModule,
		Referral:     referralService,
		Admin:        adminModule,
		Webhook:      webhookModule,
	})

	e := echo.New()
	e.HideBanner = true
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(appmw.SecurityHeaders())
	e.Use(echoMiddleware.CORSWithConfig(appmw.CORS(cfg)))

	handler.RegisterRoutes(e, h, store, appCache)
	startCron(ctx, balanceService)

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		e.Logger.Fatal(err)
	}
}

func migrationPath() string {
	if value := os.Getenv("MIGRATIONS_PATH"); value != "" {
		return value
	}
	candidates := []string{
		"internal/database/migrations",
		"migrations",
		"/app/migrations",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return "internal/database/migrations"
}

func startCron(ctx context.Context, balance *service.BalanceService) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now().In(time.FixedZone("WIB", 7*3600))
				if now.Day() == 1 {
					_ = balance.DeductMonthly(context.Background())
				}
				_ = balance.CheckAndSuspend(context.Background())
				_ = balance.SendExpiryReminders(context.Background())
			}
		}
	}()
}

// @title           Money Tracker API
// @version         1.0
// @description     REST API untuk aplikasi pencatat keuangan pribadi.
// @host            localhost:8080
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Masukkan token dengan format: Bearer {token}

package main

import (
	_ "money-tracker-service/docs"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"money-tracker-service/internal/cache"
	"money-tracker-service/internal/config"
	"money-tracker-service/internal/database"
	"money-tracker-service/internal/handler"
	appmw "money-tracker-service/internal/middleware"
	accountsmodule "money-tracker-service/internal/modules/accounts"
	adminmodule "money-tracker-service/internal/modules/admin"
	authmodule "money-tracker-service/internal/modules/auth"
	balancemodule "money-tracker-service/internal/modules/balance"
	budgetmodule "money-tracker-service/internal/modules/budget"
	categoriesmodule "money-tracker-service/internal/modules/categories"
	dashboardmodule "money-tracker-service/internal/modules/dashboard"
	groupsmodule "money-tracker-service/internal/modules/groups"
	paymentsmodule "money-tracker-service/internal/modules/payments"
	referralmodule "money-tracker-service/internal/modules/referral"
	tokensmodule "money-tracker-service/internal/modules/tokens"
	transactions "money-tracker-service/internal/modules/transactions"
	usersmodule "money-tracker-service/internal/modules/users"
	webhookmodule "money-tracker-service/internal/modules/webhook"
)

func main() {
	_ = godotenv.Load()
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

	appCache := cache.New(redisClient)

	authModule := authmodule.NewModule(cfg, db, appCache)
	authService := authModule.Service
	userModule := usersmodule.NewModule(db, appCache)
	balanceModule := balancemodule.NewModule(db)
	balanceService := balanceModule.Service
	parser := transactions.NewSmartParser(cfg, appCache)
	fonnte := webhookmodule.NewFonnteClient(cfg)
	paymentModule := paymentsmodule.NewModule(db, appCache)
	tokenModule := tokensmodule.NewModule(db)
	accountModule := accountsmodule.NewModule(db)
	transactionModule := transactions.NewModule(db, appCache, parser, accountModule.Repository)
	transactionService := transactionModule.Service
	dashboardModule := dashboardmodule.NewModule(appCache, db)
	groupModule := groupsmodule.NewModule(db, appCache, transactionService)
	referralModule := referralmodule.NewModule(cfg, db)
	adminModule := adminmodule.NewModule(authService, paymentModule.Service, adminmodule.NewRepository(db), appCache)
	webhookModule := webhookmodule.NewModule(cfg, db, appCache, parser, fonnte, transactionService)
	budgetModule := budgetmodule.NewModule(db)
	categoryModule := categoriesmodule.NewModule(db)

	authModule.Service.SetCategorySeeder(categoryModule.Service)
	authModule.Service.SetReferralCreator(referralModule.Service)
	transactionModule.Service.SetCategoryValidator(categoryModule.Service)

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
		Accounts:     accountModule,
		Dashboard:    dashboardModule,
		Groups:       groupModule,
		Referral:     referralModule,
		Admin:        adminModule,
		Webhook:      webhookModule,
		Budget:       budgetModule,
		Categories:   categoryModule,
	})

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = handler.ErrorHandler
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(appmw.SecurityHeaders())
	e.Use(echoMiddleware.CORSWithConfig(appmw.CORS(cfg)))

	handler.RegisterRoutes(e, h, appCache)
	handler.RegisterDocsRoutes(e)
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

func startCron(ctx context.Context, balance *balancemodule.Service) {
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

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	authinfra "github.com/trvux/elc-go/internal/auth/infrastructure"
	authpresentation "github.com/trvux/elc-go/internal/auth/presentation"
	branchinfra "github.com/trvux/elc-go/internal/branch/infrastructure"
	branchpresentation "github.com/trvux/elc-go/internal/branch/presentation"
	brandinfra "github.com/trvux/elc-go/internal/brand/infrastructure"
	brandpresentation "github.com/trvux/elc-go/internal/brand/presentation"
	catalogInfra "github.com/trvux/elc-go/internal/catalog/infrastructure"
	catalogPresentation "github.com/trvux/elc-go/internal/catalog/presentation"
	categoryinfra "github.com/trvux/elc-go/internal/category/infrastructure"
	categorypresentation "github.com/trvux/elc-go/internal/category/presentation"
	contactinfra "github.com/trvux/elc-go/internal/contact/infrastructure"
	contactpresentation "github.com/trvux/elc-go/internal/contact/presentation"
	eventinfra "github.com/trvux/elc-go/internal/event/infrastructure"
	eventpresentation "github.com/trvux/elc-go/internal/event/presentation"
	groupinfra "github.com/trvux/elc-go/internal/group/infrastructure"
	grouppresentation "github.com/trvux/elc-go/internal/group/presentation"
	inquirydomain "github.com/trvux/elc-go/internal/inquiry/domain"
	inquiryinfra "github.com/trvux/elc-go/internal/inquiry/infrastructure"
	inquirypresentation "github.com/trvux/elc-go/internal/inquiry/presentation"
	newsinfra "github.com/trvux/elc-go/internal/news/infrastructure"
	newspresentation "github.com/trvux/elc-go/internal/news/presentation"
	pageinfra "github.com/trvux/elc-go/internal/page/infrastructure"
	pagepresentation "github.com/trvux/elc-go/internal/page/presentation"
	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/logger"
	projecttypeinfra "github.com/trvux/elc-go/internal/project-type/infrastructure"
	projecttypepresentation "github.com/trvux/elc-go/internal/project-type/presentation"
	projectinfra "github.com/trvux/elc-go/internal/project/infrastructure"
	projectpresentation "github.com/trvux/elc-go/internal/project/presentation"
	servicegroupinfra "github.com/trvux/elc-go/internal/service-group/infrastructure"
	servicegrouppresentation "github.com/trvux/elc-go/internal/service-group/presentation"
	serviceinfra "github.com/trvux/elc-go/internal/service/infrastructure"
	servicepresentation "github.com/trvux/elc-go/internal/service/presentation"
	settingsinfra "github.com/trvux/elc-go/internal/settings/infrastructure"
	settingspresentation "github.com/trvux/elc-go/internal/settings/presentation"
	slugregistryinfra "github.com/trvux/elc-go/internal/slug-registry/infrastructure"
	slugregistrypresentation "github.com/trvux/elc-go/internal/slug-registry/presentation"
	systempageinfra "github.com/trvux/elc-go/internal/system-page/infrastructure"
	systempagepresentation "github.com/trvux/elc-go/internal/system-page/presentation"
)

func main() {
	// Ignored on purpose: in production there is no .env file, real env vars
	// are set by Docker/the process manager instead — that's not an error.
	_ = godotenv.Load()

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	log, err := logger.New(env)
	if err != nil {
		panic(err)
	}
	defer log.Sync()
	httpserver.SetLogger(log)

	ctx := context.Background()

	// bgCtx scopes long-lived background goroutines (currently just the Zalo
	// token refresher) — cancelled alongside the HTTP server's own shutdown
	// below so nothing keeps running after the process is told to stop.
	bgCtx, cancelBg := context.WithCancel(context.Background())
	defer cancelBg()

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	router := httpserver.New(log)

	authUserRepo := authinfra.NewPostgresUserRepository(pool)
	authTokenRepo := authinfra.NewPostgresVerificationTokenRepository(pool)
	authSessionRepo := authinfra.NewPostgresSessionRepository(pool)
	authHasher := authinfra.NewBcryptPasswordHasher()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	accessTokenTTL := 15 * time.Minute
	tokenIssuer := authinfra.NewJWTTokenIssuer(jwtSecret, accessTokenTTL)

	adminBaseURL := os.Getenv("ADMIN_BASE_URL")
	var emailSender authdomain.EmailSender
	if smtpHost := os.Getenv("SMTP_HOST"); smtpHost != "" {
		emailSender = authinfra.NewSMTPEmailSender(
			smtpHost, os.Getenv("SMTP_PORT"), os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"),
			os.Getenv("SMTP_FROM"), adminBaseURL,
		)
	} else {
		log.Warn("SMTP_HOST not set — invite/reset emails will be logged instead of sent")
		emailSender = authinfra.NewLogEmailSender(log, adminBaseURL)
	}

	authHandler := authpresentation.NewAuthHandler(
		authUserRepo, authTokenRepo, authSessionRepo, authHasher, tokenIssuer, emailSender,
		accessTokenTTL, env == "production",
	)
	authpresentation.RegisterRoutes(router, authHandler, tokenIssuer)

	contactRepo := contactinfra.NewPostgresContactRepository(pool)
	contactHandler := contactpresentation.NewContactHandler(contactRepo)
	contactpresentation.RegisterRoutes(router, contactHandler, tokenIssuer)

	inquiryRepo := inquiryinfra.NewPostgresInquiryRepository(pool)
	zaloFollowerRepo := inquiryinfra.NewPostgresZaloFollowerRepository(pool)
	zaloTokenRepo := inquiryinfra.NewPostgresZaloTokenRepository(pool)

	// leadNotifier falls back to LogLeadNotifier until Zalo OA credentials
	// are set — lead capture works end-to-end regardless, staff just see
	// new leads in /admin/inquiries instead of also getting a push. The
	// webhook route is always mounted either way (see RegisterRoutes) so
	// its URL can be registered in Zalo's console at any time.
	var leadNotifier inquirydomain.LeadNotifier
	zaloAppID := os.Getenv("ZALO_OA_APP_ID")
	zaloAppSecret := os.Getenv("ZALO_OA_APP_SECRET")
	if zaloAppID != "" && zaloAppSecret != "" {
		leadNotifier = inquiryinfra.NewZaloOASender(zaloTokenRepo, zaloFollowerRepo, adminBaseURL, log)
		refresher := inquiryinfra.NewZaloTokenRefresher(zaloTokenRepo, zaloAppID, zaloAppSecret, log, 20*time.Minute)
		go refresher.Start(bgCtx)
	} else {
		log.Warn("ZALO_OA_APP_ID/ZALO_OA_APP_SECRET not set — lead notifications will be logged instead of sent via Zalo")
		leadNotifier = inquiryinfra.NewLogLeadNotifier(log)
	}

	inquiryHandler := inquirypresentation.NewInquiryHandler(inquiryRepo, leadNotifier)
	// Webhook signature verification uses the OA Secret Key (generated by
	// Zalo when the webhook URL is registered in the App console), which is
	// distinct from ZALO_OA_APP_SECRET — see zalo_webhook_verifier.go.
	zaloWebhookSecret := os.Getenv("ZALO_OA_WEBHOOK_SECRET")
	zaloWebhookHandler := inquirypresentation.NewZaloWebhookHandler(zaloFollowerRepo, zaloWebhookSecret, log)
	inquirypresentation.RegisterRoutes(router, inquiryHandler, zaloWebhookHandler, tokenIssuer)

	eventRepo := eventinfra.NewPostgresEventRepository(pool)
	eventHandler := eventpresentation.NewEventHandler(eventRepo)
	eventpresentation.RegisterRoutes(router, eventHandler)

	brandRepo := brandinfra.NewPostgresBrandRepository(pool)
	brandHandler := brandpresentation.NewBrandHandler(brandRepo)
	brandpresentation.RegisterRoutes(router, brandHandler, tokenIssuer)

	catalogRepo := catalogInfra.NewPostgresProductRepository(pool)
	catalogHandler := catalogPresentation.NewProductHandler(catalogRepo)
	catalogPresentation.RegisterRoutes(router, catalogHandler, tokenIssuer)

	branchRepo := branchinfra.NewPostgresBranchRepository(pool)
	branchHandler := branchpresentation.NewBranchHandler(branchRepo)
	branchpresentation.RegisterRoutes(router, branchHandler, tokenIssuer)

	serviceGroupRepo := servicegroupinfra.NewPostgresServiceGroupRepository(pool)
	serviceGroupHandler := servicegrouppresentation.NewServiceGroupHandler(serviceGroupRepo)
	servicegrouppresentation.RegisterRoutes(router, serviceGroupHandler, tokenIssuer)

	serviceRepo := serviceinfra.NewPostgresServiceRepository(pool)
	serviceHandler := servicepresentation.NewServiceHandler(serviceRepo)
	servicepresentation.RegisterRoutes(router, serviceHandler, tokenIssuer)

	groupRepo := groupinfra.NewPostgresGroupRepository(pool)
	groupHandler := grouppresentation.NewGroupHandler(groupRepo)
	grouppresentation.RegisterRoutes(router, groupHandler, tokenIssuer)

	categoryRepo := categoryinfra.NewPostgresCategoryRepository(pool)
	categoryHandler := categorypresentation.NewCategoryHandler(categoryRepo)
	categorypresentation.RegisterRoutes(router, categoryHandler, tokenIssuer)

	settingsRepo := settingsinfra.NewPostgresSettingsRepository(pool)
	settingsHandler := settingspresentation.NewSettingsHandler(settingsRepo)
	settingspresentation.RegisterRoutes(router, settingsHandler, tokenIssuer)

	pageRepo := pageinfra.NewPostgresPageRepository(pool)
	pageHandler := pagepresentation.NewPageHandler(pageRepo)
	pagepresentation.RegisterRoutes(router, pageHandler, tokenIssuer)

	projectRepo := projectinfra.NewPostgresProjectRepository(pool)
	projectHandler := projectpresentation.NewProjectHandler(projectRepo)
	projectpresentation.RegisterRoutes(router, projectHandler, tokenIssuer)

	projectTypeRepo := projecttypeinfra.NewPostgresProjectTypeRepository(pool)
	projectTypeHandler := projecttypepresentation.NewProjectTypeHandler(projectTypeRepo)
	projecttypepresentation.RegisterRoutes(router, projectTypeHandler, tokenIssuer)

	newsRepo := newsinfra.NewPostgresNewsRepository(pool)
	newsHandler := newspresentation.NewNewsHandler(newsRepo)
	newspresentation.RegisterRoutes(router, newsHandler, tokenIssuer)

	systemPageRepo := systempageinfra.NewPostgresSystemPageRepository(pool)
	systemPageHandler := systempagepresentation.NewSystemPageHandler(systemPageRepo)
	systempagepresentation.RegisterRoutes(router, systemPageHandler, tokenIssuer)

	slugRegistryRepo := slugregistryinfra.NewPostgresSlugRegistryRepository(pool)
	slugRegistryHandler := slugregistrypresentation.NewSlugRegistryHandler(slugRegistryRepo)
	slugregistrypresentation.RegisterRoutes(router, slugRegistryHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// ListenAndServe blocks, so it runs in its own goroutine — otherwise the
	// signal-handling code below would never get a chance to run.
	go func() {
		log.Info("server starting", zap.String("port", port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	// Block the main goroutine until the OS sends SIGINT (Ctrl+C) or SIGTERM
	// (what `docker stop` sends) — this is what makes shutdown graceful
	// instead of the process being killed mid-request.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
}

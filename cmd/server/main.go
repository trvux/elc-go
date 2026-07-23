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

	attributeinfra "github.com/trvux/elc-go/internal/attribute/infrastructure"
	attributepresentation "github.com/trvux/elc-go/internal/attribute/presentation"
	authdomain "github.com/trvux/elc-go/internal/auth/domain"
	authinfra "github.com/trvux/elc-go/internal/auth/infrastructure"
	authpresentation "github.com/trvux/elc-go/internal/auth/presentation"
	authorinfra "github.com/trvux/elc-go/internal/author/infrastructure"
	authorpresentation "github.com/trvux/elc-go/internal/author/presentation"
	branchinfra "github.com/trvux/elc-go/internal/branch/infrastructure"
	branchpresentation "github.com/trvux/elc-go/internal/branch/presentation"
	brandinfra "github.com/trvux/elc-go/internal/brand/infrastructure"
	brandpresentation "github.com/trvux/elc-go/internal/brand/presentation"
	categoryinfra "github.com/trvux/elc-go/internal/category/infrastructure"
	categorypresentation "github.com/trvux/elc-go/internal/category/presentation"
	chatloginfra "github.com/trvux/elc-go/internal/chat-log/infrastructure"
	chatlogpresentation "github.com/trvux/elc-go/internal/chat-log/presentation"
	contactinfra "github.com/trvux/elc-go/internal/contact/infrastructure"
	contactpresentation "github.com/trvux/elc-go/internal/contact/presentation"
	eventinfra "github.com/trvux/elc-go/internal/event/infrastructure"
	eventpresentation "github.com/trvux/elc-go/internal/event/presentation"
	groupinfra "github.com/trvux/elc-go/internal/group/infrastructure"
	grouppresentation "github.com/trvux/elc-go/internal/group/presentation"
	inquiryinfra "github.com/trvux/elc-go/internal/inquiry/infrastructure"
	inquirypresentation "github.com/trvux/elc-go/internal/inquiry/presentation"
	newsinfra "github.com/trvux/elc-go/internal/news/infrastructure"
	newspresentation "github.com/trvux/elc-go/internal/news/presentation"
	pageinfra "github.com/trvux/elc-go/internal/page/infrastructure"
	pagepresentation "github.com/trvux/elc-go/internal/page/presentation"
	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/logger"
	productInfra "github.com/trvux/elc-go/internal/product/infrastructure"
	productPresentation "github.com/trvux/elc-go/internal/product/presentation"
	projecttypeinfra "github.com/trvux/elc-go/internal/project-type/infrastructure"
	projecttypepresentation "github.com/trvux/elc-go/internal/project-type/presentation"
	projectinfra "github.com/trvux/elc-go/internal/project/infrastructure"
	projectpresentation "github.com/trvux/elc-go/internal/project/presentation"
	recentlyviewedinfra "github.com/trvux/elc-go/internal/recently-viewed/infrastructure"
	recentlyviewedpresentation "github.com/trvux/elc-go/internal/recently-viewed/presentation"
	reviewinfra "github.com/trvux/elc-go/internal/review/infrastructure"
	reviewpresentation "github.com/trvux/elc-go/internal/review/presentation"
	servicegroupinfra "github.com/trvux/elc-go/internal/service-group/infrastructure"
	servicegrouppresentation "github.com/trvux/elc-go/internal/service-group/presentation"
	serviceinfra "github.com/trvux/elc-go/internal/service/infrastructure"
	servicepresentation "github.com/trvux/elc-go/internal/service/presentation"
	settingsinfra "github.com/trvux/elc-go/internal/settings/infrastructure"
	settingspresentation "github.com/trvux/elc-go/internal/settings/presentation"
	shippingzoneinfra "github.com/trvux/elc-go/internal/shippingzone/infrastructure"
	shippingzonepresentation "github.com/trvux/elc-go/internal/shippingzone/presentation"
	slugregistryinfra "github.com/trvux/elc-go/internal/slug-registry/infrastructure"
	slugregistrypresentation "github.com/trvux/elc-go/internal/slug-registry/presentation"
	systempageinfra "github.com/trvux/elc-go/internal/system-page/infrastructure"
	systempagepresentation "github.com/trvux/elc-go/internal/system-page/presentation"
	taginfra "github.com/trvux/elc-go/internal/tag/infrastructure"
	tagpresentation "github.com/trvux/elc-go/internal/tag/presentation"
	uploaddomain "github.com/trvux/elc-go/internal/upload/domain"
	uploadinfra "github.com/trvux/elc-go/internal/upload/infrastructure"
	uploadpresentation "github.com/trvux/elc-go/internal/upload/presentation"
	wishlistinfra "github.com/trvux/elc-go/internal/wishlist/infrastructure"
	wishlistpresentation "github.com/trvux/elc-go/internal/wishlist/presentation"
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
	inquiryHandler := inquirypresentation.NewInquiryHandler(inquiryRepo)
	inquirypresentation.RegisterRoutes(router, inquiryHandler, tokenIssuer)

	reviewRepo := reviewinfra.NewPostgresReviewRepository(pool)
	reviewHandler := reviewpresentation.NewReviewHandler(reviewRepo)
	reviewpresentation.RegisterRoutes(router, reviewHandler, tokenIssuer)

	eventRepo := eventinfra.NewPostgresEventRepository(pool)
	eventHandler := eventpresentation.NewEventHandler(eventRepo)
	eventpresentation.RegisterRoutes(router, eventHandler)

	brandRepo := brandinfra.NewPostgresBrandRepository(pool)
	brandHandler := brandpresentation.NewBrandHandler(brandRepo)
	brandpresentation.RegisterRoutes(router, brandHandler, tokenIssuer)

	authorRepo := authorinfra.NewPostgresAuthorRepository(pool)
	authorHandler := authorpresentation.NewAuthorHandler(authorRepo)
	authorpresentation.RegisterRoutes(router, authorHandler, tokenIssuer)

	tagRepo := taginfra.NewPostgresTagRepository(pool)
	tagHandler := tagpresentation.NewTagHandler(tagRepo)
	tagpresentation.RegisterRoutes(router, tagHandler, tokenIssuer)

	attributeDefinitionRepo := attributeinfra.NewPostgresAttributeDefinitionRepository(pool)
	attributeDefinitionHandler := attributepresentation.NewAttributeDefinitionHandler(attributeDefinitionRepo)
	attributepresentation.RegisterRoutes(router, attributeDefinitionHandler, tokenIssuer)

	productRepo := productInfra.NewPostgresProductRepository(pool)
	productHandler := productPresentation.NewProductHandler(productRepo, attributeDefinitionRepo)
	productPresentation.RegisterRoutes(router, productHandler, tokenIssuer)

	productLineRepo := productInfra.NewPostgresProductLineRepository(pool)
	productLineHandler := productPresentation.NewProductLineHandler(productLineRepo)
	productPresentation.RegisterProductLineRoutes(router, productLineHandler, tokenIssuer)

	catalogPageRepo := productInfra.NewPostgresCatalogPageRepository(pool)
	catalogPageHandler := productPresentation.NewCatalogPageHandler(catalogPageRepo)
	productPresentation.RegisterCatalogPageRoutes(router, catalogPageHandler, tokenIssuer)

	// secureCookies also gates the wishlist/recently-viewed/chat-log
	// visitor_id cookie's Secure flag — same production-only rule as
	// auth's refresh_token cookie above.
	secureCookies := env == "production"

	wishlistRepo := wishlistinfra.NewPostgresWishlistRepository(pool)
	wishlistHandler := wishlistpresentation.NewWishlistHandler(wishlistRepo)
	wishlistpresentation.RegisterRoutes(router, wishlistHandler, secureCookies)

	recentlyViewedRepo := recentlyviewedinfra.NewPostgresRecentlyViewedRepository(pool)
	recentlyViewedHandler := recentlyviewedpresentation.NewRecentlyViewedHandler(recentlyViewedRepo)
	recentlyviewedpresentation.RegisterRoutes(router, recentlyViewedHandler, secureCookies)

	chatLogRepo := chatloginfra.NewPostgresChatLogRepository(pool)
	chatLogHandler := chatlogpresentation.NewChatLogHandler(chatLogRepo)
	chatlogpresentation.RegisterRoutes(router, chatLogHandler, tokenIssuer, secureCookies)

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

	shippingZoneRepo := shippingzoneinfra.NewPostgresShippingZoneRepository(pool)
	provinceRepo := shippingzoneinfra.NewPostgresProvinceRepository(pool)
	wardRepo := shippingzoneinfra.NewPostgresWardRepository(pool)
	shippingZoneHandler := shippingzonepresentation.NewShippingZoneHandler(shippingZoneRepo, provinceRepo, wardRepo)
	shippingzonepresentation.RegisterRoutes(router, shippingZoneHandler, tokenIssuer)

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

	var uploader uploaddomain.Uploader
	if r2AccountID := os.Getenv("R2_ACCOUNT_ID"); r2AccountID != "" {
		uploader = uploadinfra.NewR2Uploader(
			r2AccountID, os.Getenv("R2_ACCESS_KEY_ID"), os.Getenv("R2_SECRET_ACCESS_KEY"),
			os.Getenv("R2_BUCKET_NAME"), os.Getenv("R2_PUBLIC_URL"),
		)
	} else {
		log.Warn("R2_ACCOUNT_ID not set — image uploads will fail until R2 is configured")
		uploader = uploadinfra.NewNoopUploader()
	}
	uploadHandler := uploadpresentation.NewUploadHandler(uploader, uploadinfra.NewImageCropper())
	uploadpresentation.RegisterRoutes(router, uploadHandler, tokenIssuer)

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

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

	brandinfra "github.com/trvux/elc-go/internal/brand/infrastructure"
	brandpresentation "github.com/trvux/elc-go/internal/brand/presentation"
	branchinfra "github.com/trvux/elc-go/internal/branch/infrastructure"
	branchpresentation "github.com/trvux/elc-go/internal/branch/presentation"
	contactinfra "github.com/trvux/elc-go/internal/contact/infrastructure"
	contactpresentation "github.com/trvux/elc-go/internal/contact/presentation"
	groupinfra "github.com/trvux/elc-go/internal/group/infrastructure"
	grouppresentation "github.com/trvux/elc-go/internal/group/presentation"
	"github.com/trvux/elc-go/internal/platform/db"
	"github.com/trvux/elc-go/internal/platform/httpserver"
	"github.com/trvux/elc-go/internal/platform/logger"
	servicegroupinfra "github.com/trvux/elc-go/internal/service-group/infrastructure"
	servicegrouppresentation "github.com/trvux/elc-go/internal/service-group/presentation"
	serviceinfra "github.com/trvux/elc-go/internal/service/infrastructure"
	servicepresentation "github.com/trvux/elc-go/internal/service/presentation"
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

	contactRepo := contactinfra.NewPostgresContactRepository(pool)
	contactHandler := contactpresentation.NewContactHandler(contactRepo)
	contactpresentation.RegisterRoutes(router, contactHandler)

	brandRepo := brandinfra.NewPostgresBrandRepository(pool)
	brandHandler := brandpresentation.NewBrandHandler(brandRepo)
	brandpresentation.RegisterRoutes(router, brandHandler)

	branchRepo := branchinfra.NewPostgresBranchRepository(pool)
	branchHandler := branchpresentation.NewBranchHandler(branchRepo)
	branchpresentation.RegisterRoutes(router, branchHandler)

	serviceGroupRepo := servicegroupinfra.NewPostgresServiceGroupRepository(pool)
	serviceGroupHandler := servicegrouppresentation.NewServiceGroupHandler(serviceGroupRepo)
	servicegrouppresentation.RegisterRoutes(router, serviceGroupHandler)

	serviceRepo := serviceinfra.NewPostgresServiceRepository(pool)
	serviceHandler := servicepresentation.NewServiceHandler(serviceRepo)
	servicepresentation.RegisterRoutes(router, serviceHandler)

	groupRepo := groupinfra.NewPostgresGroupRepository(pool)
	groupHandler := grouppresentation.NewGroupHandler(groupRepo)
	grouppresentation.RegisterRoutes(router, groupHandler)

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

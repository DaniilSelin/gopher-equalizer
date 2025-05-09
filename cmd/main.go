package main

import (
	"fmt"
	"os"
	"os/signal"
	"context"
	"log"
	"time"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"

	"gopher-equalizer/config"
	"gopher-equalizer/internal/logger"
	"gopher-equalizer/internal/database"
	"gopher-equalizer/internal/repository"
	"gopher-equalizer/internal/service"
	"gopher-equalizer/internal/balancer"
	"gopher-equalizer/internal/transport/http/api"
	"gopher-equalizer/internal/transport/http/proxy"
    health "gopher-equalizer/internal/transport/http"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	srv, dbPool, err := run(ctx, os.Stdout, os.Args);
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	<-ctx.Done()
	log.Println("Shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        log.Fatalf("Server shutdown failed: %v", err)
    }
    log.Println("Server exited gracefully")
}

func run(ctx context.Context, w io.Writer, args []string) (*http.Server, *pgxpool.Pool, error) {
    // 1. Конфиг и логгер
    cfg, err := config.LoadConfig("config/config.yml")
    if err != nil {
        return nil, nil, err
    }
    ctx, err = logger.New(ctx, cfg)
    if err != nil {
        return nil, nil, err
    }
    log := logger.GetLoggerFromCtx(ctx)

    // 2. Подключение к БД и миграции
    dbPool, err := database.Connect(ctx, cfg)
    if err != nil {
        return nil, nil, err
    }
    if err := database.RunMigrations(ctx, cfg, dbPool); err != nil {
        return nil, nil, err
    }

    // 3. Репозиторий и bucket-сервис
    repo := repository.NewBucketRepository(dbPool, cfg)
    bSrv := service.NewBucketService(cfg, repo)

    // 4. HTTP-API для управления buckets
    apiH := api.NewHandler(ctx, cfg, bSrv)
    apiMux := api.NewRouter(apiH)

    // 5. Балансировщик и прокси
    strat, err := balancer.CreateStrategy(cfg.Balancer.Strategy, cfg.Balancer.Backends)
    if err != nil {
        return nil, nil, err
    }
    bal := balancer.NewBalancer(strat)
    healcheck := health.NewHealthChecker(cfg, bal)

    // 6. Запускаем хелф-чекер
    healcheck.StartHealthChecks(ctx)

    proxy := proxy.NewProxy(cfg, bal, bSrv, log)

    // 7. Общий mux: сначала API, потом прокси «на всё остальное»
    mux := http.NewServeMux()
    mux.Handle("/buckets", apiMux)
    mux.Handle("/buckets/", apiMux)
    mux.Handle("/", proxy)

    // 8. HTTP-сервер
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    srv := &http.Server{
        Addr:    addr,
        Handler: mux,
    }
    log.Info(ctx, "starting server", zap.String("addr", addr))
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Error(ctx, "ListenAndServe failed", zap.Error(err))
        }
    }()

    return srv, dbPool, nil
}

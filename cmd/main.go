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

	"gopher-equalizer/config"
	"gopher-equalizer/internal/logger"
	"gopher-equalizer/internal/database"
	"gopher-equalizer/internal/repository"
	"gopher-equalizer/internal/service"
	"gopher-equalizer/internal/transport/http/api"
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
	// Загружаем конфиг
	cfg, err := config.LoadConfig("config/config.yml")
	if err != nil {
		return nil, nil, fmt.Errorf("Error loading config: %v", err)
	}

	// Подключаем логгирование
	ctx, err = logger.New(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("Error create logger: %v", err)
	}

	//Подключаемся к БД
	dbPool, err := database.Connect(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("Database connection failed: %v", err)
	}


	// Запускаем миграции
	err = database.RunMigrations(ctx, cfg, dbPool)
	if err != nil {
		return nil, nil, fmt.Errorf("Migration failed: %v", err)
	}

	// Создаем репозитории
	repo := repository.NewBucketRepository(dbPool, cfg)

	bSrv := service.NewBucketService(cfg, repo)

	// Создаем хэндлер
	handler := api.NewHandler(ctx, cfg, bSrv)

	// Создаём роутер
	router := api.NewRouter(handler)

	//Запускаем сервер
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting server on %s...", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return srv, dbPool, nil
}
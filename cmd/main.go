package main

import (
	"fmt"
	"os"
	"os/signal"
	"context"
	"log"
	"io"

	 _ "github.com/davecgh/go-spew/spew"

	"gopher-equalizer/config"
	"gopher-equalizer/internal/logger"
	"gopher-equalizer/internal/database"
	"gopher-equalizer/internal/repository"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, w io.Writer, args []string) error {
	// Загружаем конфиг
	cfg, err := config.LoadConfig("config/config.yml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Подключаем логгирование
	ctx, err = logger.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Error create logger: %v", err)
	}

	//Подключаемся к БД
	dbPool, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbPool.Close()

	// Запускаем миграции
	err = database.RunMigrations(ctx, cfg, dbPool)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Создаем репозитории
	_ = repository.NewBucketRepository(dbPool, cfg)

	return nil
}
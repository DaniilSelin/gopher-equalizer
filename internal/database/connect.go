package database

import (
	"gopher-equalizer/config"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

func Connect(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	// создание конфига
	poolConfig, err := pgxpool.ParseConfig(cfg.DB.ConnString())
	if err != nil {
		return nil, err
	}
	// Сюда же можно определить - BeforeConnect, AfterRelease и подобное
	poolConfig.MaxConns = cfg.DB.Pool.MaxConns
	poolConfig.MinConns = cfg.DB.Pool.MinConns
	poolConfig.MaxConnLifetime = time.Duration(cfg.DB.Pool.MaxConnLifetime)
	poolConfig.MaxConnIdleTime = time.Duration(cfg.DB.Pool.MaxConnIdleTime)
	poolConfig.HealthCheckPeriod = time.Duration(cfg.DB.Pool.HealthCheckPeriod)
	// создание пула
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		pool, err = Retry(poolConfig,
			cfg.DB.ConnectRetries,
			time.Duration(cfg.DB.ConnectRetryDelay),
		)
		if err != nil {
			return nil, err
		}
	}
	return pool, nil
}

func Retry(poolConfig *pgxpool.Config, retries int, delay time.Duration) (*pgxpool.Pool, error) {
	var err error
	for i := 0; i < retries; i++ {
		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			return pool, nil
		}
		time.Sleep(delay)
	}
	return nil, err
}

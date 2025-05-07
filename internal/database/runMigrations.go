package database

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopher-equalizer/config"
	"gopher-equalizer/internal/errdefs"

	"github.com/jackc/pgx/v5/pgxpool"
)

var MigrationPath = "internal/database/migrations"

func RunMigrations(ctx context.Context, cfg *config.Config, conn *pgxpool.Pool) error {

	files, err := os.ReadDir(MigrationPath)
	if err != nil {
		return fmt.Errorf("%w: could not read migrations dir: %v", errdefs.ErrMigrationFailed, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		path := filepath.Join(MigrationPath, file.Name())
		sqlContent, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read SQL file %s: %w", path, err)
		}

		sqlQuery := fmt.Sprintf(string(sqlContent), cfg.DB.Schema, cfg.DB.Schema)

		_, err = conn.Exec(ctx, sqlQuery)
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", path, err)
		}

		log.Printf("Successfully executed migration: %s", path)
	}
	return nil
}

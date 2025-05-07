package repository

import (
	"context"

	"gopher-equalizer/config"
	"gopher-equalizer/internal/errdefs"
	"gopher-equalizer/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	chkTokensNonNeg = "ck_tokens_nonnegative"
	chkTokensLeCap  = "ck_tokens_le_capacity"
)

type BucketRepository struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewBucketRepository(db *pgxpool.Pool, cfg *config.Config) BucketRepository {
	return BucketRepository{
		db:  db,
		cfg: cfg,
	}
}

// Логика
func (br BucketRepository) TryConsume(ctx context.Context, clientID string) error {
	query := `
	UPDATE token_buckets 
		SET tokens = tokens - 1
	WHERE client_id = $1
	`

	tag, err := br.db.Exec(ctx, query, clientID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errdefs.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case chkTokensNonNeg:
				return errdefs.NotEnoughTokens
			}
		}
		return errdefs.Wrap(errdefs.ErrDB, err.Error())
	}

	rowsAffected := tag.RowsAffected()
	if rowsAffected == 0 {
		return errdefs.ErrNotFound
	}

	return nil
}

func (br BucketRepository) RefillTokens(ctx context.Context, clientID string, amount int) error {
	query := `
	UPDATE token_buckets 
		SET tokens = LEAST(tokens + $1, capacity)
	WHERE client_id = $2
	`

	tag, err := br.db.Exec(
		ctx, query, amount, clientID)

	if err != nil {
		return errdefs.Wrap(errdefs.ErrDB, err.Error())
	}

	rowsAffected := tag.RowsAffected()
	if rowsAffected == 0 {
		return errdefs.ErrNotFound
	}

	return nil
}

// CRUD
func (br BucketRepository) CreateBucket(ctx context.Context, bucket *models.Bucket) error {
	query := `
 		INSERT INTO token_buckets (
 			client_id, capacity, tokens, last_refill
 		) VALUES ($1, $2, $3, $4)
	`
	_, err := br.db.Exec(ctx, query,
		bucket.ClientID,
		bucket.Capacity,
		bucket.Tokens,
		bucket.LastRefill,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errdefs.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				return errdefs.Wrapf(errdefs.ErrConflict, "ClientID '%s' already exists", bucket.ClientID)
			case "23514": // check_violation
				return errdefs.Wrapf(errdefs.ErrInvalidInput, "validation failed: %v", pgErr.Message)
			}
		}
		return errdefs.Wrapf(errdefs.ErrDB, "failed to create bucket: %v", err)
	}
	return nil
}

func (br BucketRepository) RemoveBucket(ctx context.Context, clientID string) error {
	query := `
		DELETE FROM token_buckets
		where client_id = $1
	`
	tag, err := br.db.Exec(ctx, query, clientID)

	if err != nil {
		return errdefs.Wrapf(errdefs.ErrDB, "failed to delete bucket %q: %v", clientID, err)
	}

	rowsAffected := tag.RowsAffected()
	if rowsAffected == 0 {
		return errdefs.ErrNotFound
	}

	return nil
}

func (br BucketRepository) UpdateCapacity(ctx context.Context, clientID string, newCapacity int) error {
	query := `
	UPDATE token_buckets 
	SET 
	    capacity = $1
	WHERE client_id = $2
	`
	tag, err := br.db.Exec(ctx, query, newCapacity, clientID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errdefs.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case chkTokensLeCap:
				return errdefs.TokensLeCap
			}
		}
		return errdefs.Wrapf(errdefs.ErrDB, "failed to update capacity buckets: %v", err)
	}

	rowsAffected := tag.RowsAffected()
	if rowsAffected == 0 {
		return errdefs.ErrNotFound
	}

	return nil
}

func (br BucketRepository) UpdateCountTokens(ctx context.Context, clientID string, newCountT int) error {
	query := `
	UPDATE token_buckets 
	SET 
	    tokens = $1
	WHERE client_id = $2
	`
	tag, err := br.db.Exec(ctx, query, newCountT, clientID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errdefs.As(err, &pgErr) {
			switch pgErr.ConstraintName {
			case chkTokensLeCap:
				return errdefs.TokensLeCap
			}
		}
		return errdefs.Wrapf(errdefs.ErrDB, "failed to update tokens buckets: %v", err)
	}

	rowsAffected := tag.RowsAffected()
	if rowsAffected == 0 {
		return errdefs.ErrNotFound
	}

	return nil
}

func (br BucketRepository) GetBucket(ctx context.Context, clientID string) (*models.Bucket, error) {
	query := `
		SELECT client_id, capacity, tokens, last_refill
		FROM token_buckets
		where client_id = $1
	`
	var bucket models.Bucket

	err := br.db.QueryRow(ctx, query, clientID).Scan(
		&bucket.ClientID,
		&bucket.Capacity,
		&bucket.Tokens,
		&bucket.LastRefill,
	)
	if err != nil {
		if errdefs.Is(err, pgx.ErrNoRows) {
			return nil, errdefs.ErrNotFound
		}
		return nil, errdefs.Wrapf(errdefs.ErrDB, "failed to get bucket %s: %v", clientID, err)
	}

	return &bucket, nil
}

// Были идеи о keyset-плагинации, но я ни разу её не реализовывал, а времени мало...
func (br BucketRepository) ListBuckets(ctx context.Context, limit, offset int) (*[]models.Bucket, error) {
	query := `
		SELECT client_id, capacity, tokens, last_refill
		FROM token_buckets
		ORDER BY last_refill DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := br.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrDB, "failed to list buckets: %v", err)
	}
	defer rows.Close()

	var buckets []models.Bucket
	for rows.Next() {
		var bucket models.Bucket
		if err := rows.Scan(&bucket.ClientID, &bucket.Capacity, &bucket.Tokens, &bucket.LastRefill); err != nil {
			return nil, errdefs.Wrapf(errdefs.ErrDB, "failed to scan bucket: %v", err)
		}
		buckets = append(buckets, bucket)
	}

	if rows.Err() != nil {
		return nil, errdefs.Wrapf(errdefs.ErrDB, "rows iteration error: %v", rows.Err())
	}

	return &buckets, nil
}

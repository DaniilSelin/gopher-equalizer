package interfaces

import (
	"context"

	"gopher-equalizer/internal/models"
)

type IBucketRepository interface {
	// CRUD
	CreateBucket(ctx context.Context, bucket *models.Bucket) error
	RemoveBucket(ctx context.Context, clientID string) error
	UpdateCapacity(ctx context.Context, clientID string, newCapacity int) error
	UpdateCountTokens(ctx context.Context, clientID string, newCountT int) error
	ListBuckets(ctx context.Context, limit, offset int) (*[]models.Bucket, error)
	GetBucket(ctx context.Context, clientID string) (*models.Bucket, error)
	// Логика
	TryConsume(ctx context.Context, clientID string) error
	RefillTokens(ctx context.Context, clientID string, amount int) error
}
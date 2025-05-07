package interfaces

import (
    "context"

    "gopher-equalizer/internal/models"
)

type IBucketService interface {
    CreateBucket(ctx context.Context, b *models.Bucket) error
    RemoveBucket(ctx context.Context, clientID string) error
    UpdateCapacity(ctx context.Context, clientID string, newCap int) error
    UpdateTokens(ctx context.Context, clientID string, newTokens int) error
    GetBucket(ctx context.Context, clientID string) (*models.Bucket, error)
    ListBuckets(ctx context.Context, limit, offset int) (*[]models.Bucket, error)
}
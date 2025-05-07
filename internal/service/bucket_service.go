package service

import (
    "context"
    "time"

    "gopher-equalizer/internal/interfaces"
    "gopher-equalizer/internal/logger"
    "gopher-equalizer/internal/models"
    "gopher-equalizer/internal/errdefs"
    "gopher-equalizer/config"

    "go.uber.org/zap"
)

type BucketService struct {
    repo interfaces.IBucketRepository
    cfg *config.Config
}

func NewBucketService(cfg *config.Config, repo interfaces.IBucketRepository) BucketService {
    return BucketService{
        repo: repo, 
        cfg: cfg,
    }
}

// Логика
func (bs BucketService) TryConsume(ctx context.Context, clientID string) error {
    logger := logger.GetLoggerFromCtx(ctx)

    err := bs.repo.TryConsume(ctx, clientID)
    if err != nil {
        if errdefs.Is(err, errdefs.ErrNotFound) {
            logger.Info(ctx, "creating new token bucket for client", zap.String("clientID", clientID))
            return bs.repo.CreateBucket(ctx,
                &models.Bucket{
                    ClientID:   clientID,
                    Capacity:   bs.cfg.Bucket.Capacity,
                    Tokens:     bs.cfg.Bucket.Capacity,
                    LastRefill: time.Now(),
                },
            )
        }
        logger.Error(ctx, "failed to consume token", zap.String("clientID", clientID), zap.Error(err))
        return err
    }
    logger.Info(ctx, "token consumed", zap.String("clientID", clientID))

    bucket, err := bs.repo.GetBucket(ctx, clientID)
    if err != nil {
        logger.Error(ctx, "failed to fetch bucket after consume", zap.String("clientID", clientID), zap.Error(err))
        return err
    }

    now := time.Now()
    elapsed := now.Sub(bucket.LastRefill)
    interval := time.Duration(bs.cfg.Bucket.Refill.Interval)
    if elapsed >= interval {
        // Сколько шагов интервала прошло
        steps := int(elapsed / interval)
        amount := steps * bs.cfg.Bucket.Refill.Amount

        logger.Info(ctx, "refilling tokens",
                    zap.String("clientID", clientID),
                    zap.Int("amount", amount),
                    zap.Int("capacity", bucket.Capacity),
                )
        if err := bs.repo.RefillTokens(ctx, clientID, amount); err != nil {
            return err
        }
        logger.Info(ctx, "tokens refilled", zap.String("clientID", clientID))
    }
    return nil
}

// CRUD
func (bs BucketService) CreateBucket(ctx context.Context, b *models.Bucket) error {
    if b.ClientID == "" {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "ClientID reqiured")
    }
    if b.Capacity <= 0 {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "Capacity must be not negative")
    }
    if b.Tokens < 0 || b.Tokens > b.Capacity {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "Tokens must be in the range [0, Capacity]")
    }
    return bs.repo.CreateBucket(ctx, b)
}

func (bs BucketService) RemoveBucket(ctx context.Context, clientID string) error {
    if clientID == "" {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "ClientID reqiured")
    }
    return bs.repo.RemoveBucket(ctx, clientID)
}

func (bs BucketService) UpdateCapacity(ctx context.Context, clientID string, newCap int) error {
    if clientID == "" {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "ClientID reqiured")
    }
    if newCap <= 0 {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "Capacity must be not negative")
    }
    return bs.repo.UpdateCapacity(ctx, clientID, newCap)
}

func (bs BucketService) UpdateTokens(ctx context.Context, clientID string, newTokens int) error {
    if clientID == "" {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "ClientID reqiured")
    }
    if newTokens < 0 {
        return errdefs.Wrap(errdefs.ErrInvalidInput, "Tokens  must be not negative")
    }
    return bs.repo.UpdateCountTokens(ctx, clientID, newTokens)
}

func (bs BucketService) GetBucket(ctx context.Context, clientID string) (*models.Bucket, error) {
    if clientID == "" {
        return nil, errdefs.Wrap(errdefs.ErrInvalidInput, "ClientID reqiured")
    }
    bucket, err := bs.repo.GetBucket(ctx, clientID)
    if err != nil {
        return nil, err
    }
    return bucket, nil
}

func (bs BucketService) ListBuckets(ctx context.Context, limit, offset int) (*[]models.Bucket, error) {
    if limit <= 0 {
        return nil, errdefs.Wrapf(errdefs.ErrInvalidInput, "limit must be not negative")
    }
    if offset < 0 {
        return nil, errdefs.Wrapf(errdefs.ErrInvalidInput, "offset must be not negative")
    }
    buckets, err := bs.repo.ListBuckets(ctx, limit, offset)
    if err != nil {
        return nil, err
    }
    return buckets, nil
}

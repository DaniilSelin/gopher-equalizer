/*
Так как код со стороны repository я покрыл тестами,
то тут его можно спокойно мокнуть и тестировать на "поведение"
*/
package service

import (
	"context"
	"testing"
    "os"
    "log"
    "time"

    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"

    "gopher-equalizer/internal/errdefs"
    "gopher-equalizer/internal/logger"
    "gopher-equalizer/internal/models"
    "gopher-equalizer/config"
)

type MockRepository struct {
    mock.Mock
}

var cfg *config.Config

func (m *MockRepository) CreateBucket(ctx context.Context, b *models.Bucket) error {
    args := m.Called(ctx, b)
    return args.Error(0)
}
func (m *MockRepository) RemoveBucket(ctx context.Context, clientID string) error {
    args := m.Called(ctx, clientID)
    return args.Error(0)
}
func (m *MockRepository) UpdateCapacity(ctx context.Context, clientID string, newCap int) error {
    args := m.Called(ctx, clientID, newCap)
    return args.Error(0)
}
func (m *MockRepository) UpdateCountTokens(ctx context.Context, clientID string, newT int) error {
    args := m.Called(ctx, clientID, newT)
    return args.Error(0)
}
func (m *MockRepository) GetBucket(ctx context.Context, clientID string) (*models.Bucket, error) {
    args := m.Called(ctx, clientID)
    return args.Get(0).(*models.Bucket), args.Error(1)
}
func (m *MockRepository) ListBuckets(ctx context.Context, limit, offset int) (*[]models.Bucket, error) {
    args := m.Called(ctx, limit, offset)
    return args.Get(0).(*[]models.Bucket), args.Error(1)
}
func (m *MockRepository) TryConsume(ctx context.Context, clientID string) error {
    args := m.Called(ctx, clientID)
    return args.Error(0)
}
func (m *MockRepository) RefillTokens(ctx context.Context, clientID string, amount int) error {
    args := m.Called(ctx, clientID, amount)
    return args.Error(0)
}

func TestMain(m *testing.M) {
    var err error

    // Загружаем конфиг
    cfg, err = config.LoadConfig("../../config/config.yml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    code := m.Run()
    os.Exit(code)
}

func TestBucketService(t *testing.T) {
    ctx := context.Background()

    t.Run("CreateBucket", func(t *testing.T) {
        mockRepo := new(MockRepository)
        svc := NewBucketService(cfg, mockRepo)

        b := &models.Bucket{
            ClientID:   "client1",
            Capacity:   10,
            Tokens:     5,
        }

        mockRepo.
            On("CreateBucket", ctx, b).
            Return(nil).
            Once()

        err := svc.CreateBucket(ctx, b)
        require.NoError(t, err)

        mockRepo.AssertExpectations(t)
    })

    t.Run("CreateBucketInvalidInput", func(t *testing.T) {
        mockRepo := new(MockRepository)
        svc := NewBucketService(cfg, mockRepo)

        // пустой ClientID -> 0 обращений к репозиторию
        err := svc.CreateBucket(ctx, &models.Bucket{ClientID: "", Capacity: 1, Tokens: 0})
        require.ErrorIs(t, err, errdefs.ErrInvalidInput)

        mockRepo.AssertNotCalled(t, "CreateBucket", mock.Anything, mock.Anything)
    })

    t.Run("DeleteBucketErr", func(t *testing.T) {
        mockRepo := new(MockRepository)
        svc := NewBucketService(cfg, mockRepo)

        mockRepo.
            On("RemoveBucket", ctx, "clientX").
            Return(errdefs.ErrNotFound).
            Once()

        err := svc.RemoveBucket(ctx, "clientX")
        require.ErrorIs(t, err, errdefs.ErrNotFound)

        mockRepo.AssertExpectations(t)
    })

    t.Run("GetBucket", func(t *testing.T) {
        mockRepo := new(MockRepository)
        svc := NewBucketService(cfg, mockRepo)

        want := &models.Bucket{ClientID: "id1", Capacity: 3, Tokens: 2}
        mockRepo.
            On("GetBucket", ctx, "id1").
            Return(want, nil).
            Once()

        got, err := svc.GetBucket(ctx, "id1")
        require.NoError(t, err)
        require.Equal(t, want, got)

        mockRepo.AssertExpectations(t)
    })

    t.Run("ListBuckets", func(t *testing.T) {
        mockRepo := new(MockRepository)
        svc := NewBucketService(cfg, mockRepo)

        list := []models.Bucket{
            {ClientID: "a", Capacity: 2, Tokens: 1},
            {ClientID: "b", Capacity: 5, Tokens: 4},
        }
        mockRepo.
            On("ListBuckets", ctx, 2, 0).
            Return(&list, nil).
            Once()

        got, err := svc.ListBuckets(ctx, 2, 0)
        require.NoError(t, err)
        require.Equal(t, &list, got)

        mockRepo.AssertExpectations(t)
    })
}

func TestTryConsume(t *testing.T) {
    ctx := context.Background()
    ctx, _ = logger.New(ctx, cfg)

   t.Run("CreatesBucketOnFirstConsume", func(t *testing.T) {
       mockRepo := new(MockRepository)
       svc := NewBucketService(cfg, mockRepo)

       mockRepo.On("GetBucket", ctx, "client1").Return((*models.Bucket)(nil), errdefs.ErrNotFound).Once()
       mockRepo.On("CreateBucket", ctx, mock.MatchedBy(func(b *models.Bucket) bool {
           return b.ClientID == "client1" && b.Tokens == cfg.Bucket.Capacity-1
       })).Return(nil).Once()

       err := svc.TryConsume(ctx, "client1")
       require.NoError(t, err)
       mockRepo.AssertExpectations(t)
   })

   t.Run("NoRefillAndConsume", func(t *testing.T) {
       mockRepo := new(MockRepository)
       svc := NewBucketService(cfg, mockRepo)

       bucket := &models.Bucket{ClientID: "c2", Capacity: 5, Tokens: 3, LastRefill: time.Now()}
       mockRepo.On("GetBucket", ctx, "c2").Return(bucket, nil).Once()
       mockRepo.On("TryConsume", ctx, "c2").Return(nil).Once()

       err := svc.TryConsume(ctx, "c2")
       require.NoError(t, err)
       mockRepo.AssertExpectations(t)
   })

   t.Run("RefillThenConsume", func(t *testing.T) {
       mockRepo := new(MockRepository)
       svc := NewBucketService(cfg, mockRepo)

       past := time.Now().Add(
            -2 * time.Duration(cfg.Bucket.Refill.Interval),
        )
       bucket := &models.Bucket{ClientID: "c3", Capacity: 10, Tokens: 1, LastRefill: past}
       mockRepo.On("GetBucket", ctx, "c3").Return(bucket, nil).Once()
       expectedAmount := 2 * cfg.Bucket.Refill.Amount
       mockRepo.On("RefillTokens", ctx, "c3", expectedAmount).Return(nil).Once()
       mockRepo.On("TryConsume", ctx, "c3").Return(nil).Once()

       err := svc.TryConsume(ctx, "c3")
       require.NoError(t, err)
       mockRepo.AssertExpectations(t)
   })

   t.Run("InsufficientTokens", func(t *testing.T) {
       mockRepo := new(MockRepository)
       svc := NewBucketService(cfg, mockRepo)

       bucket := &models.Bucket{ClientID: "c4", Capacity: 5, Tokens: 0, LastRefill: time.Now()}
       mockRepo.On("GetBucket", ctx, "c4").Return(bucket, nil).Once()
       mockRepo.On("TryConsume", ctx, "c4").Return(errdefs.NotEnoughTokens).Once()

       err := svc.TryConsume(ctx, "c4")
       require.ErrorIs(t, err, errdefs.NotEnoughTokens)
       mockRepo.AssertExpectations(t)
   })
}

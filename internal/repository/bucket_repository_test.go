package repository

import (
    "fmt"
    "context"
    "log"
    "os"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/jackc/pgx/v5/pgxpool"

    "gopher-equalizer/config"
    "gopher-equalizer/internal/models"
    "gopher-equalizer/internal/errdefs"
    "gopher-equalizer/internal/database"
)

var db *pgxpool.Pool
var cfg *config.Config

func TestMain(m *testing.M) {
    var err error

    // Загружаем конфиг
    cfg, err = config.LoadConfig("../../config/config.yml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Подключаемся к бд
    db, err = database.Connect(context.Background(), cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }

    code := m.Run()
    os.Exit(code)
}

func clearTable(t *testing.T) {
    ctx := context.Background()
    _, err := db.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s.token_buckets", cfg.DB.Schema))

    require.NoError(t, err, "Не удалось очистить таблицу buckets")
}

func TestBucketRepository(t *testing.T) {
    ctx := context.Background()
    repo := NewBucketRepository(db, cfg)

    t.Run("CreateGetRemove", func(t *testing.T) {
        clearTable(t)

        bucket := &models.Bucket{
            ClientID: "test-client-1",
            Capacity: 5,
            Tokens:   5,
        }
        err := repo.CreateBucket(ctx, bucket)
        require.NoError(t, err, "Ошибка при создании бакета")

        got, err := repo.GetBucket(ctx, bucket.ClientID)
        require.NoError(t, err, "Ошибка при получении бакета")
        require.Equal(t, bucket.ClientID, got.ClientID, "Неверный ClientID")
        require.Equal(t, bucket.Capacity, got.Capacity, "Неверная capacity")
        require.Equal(t, bucket.Tokens, got.Tokens, "Неверное число токенов")

        err = repo.RemoveBucket(ctx, bucket.ClientID)
        require.NoError(t, err, "Ошибка при удалении бакета")

        _, err = repo.GetBucket(ctx, bucket.ClientID)
        require.Equal(t, err, errdefs.ErrNotFound, "Ожидалась ошибка при получении несуществующего бакета")
    })

    t.Run("UniqueClientIDError", func(t *testing.T) {
        clearTable(t)

        bucket := &models.Bucket{
            ClientID: "duplicate-client",
            Capacity: 3,
            Tokens:   3,
        }

        err := repo.CreateBucket(ctx, bucket)
        require.NoError(t, err, "Ошибка при создании первого бакета")

        // Второе создание с тем же ClientID должно привести к ошибке
        err = repo.CreateBucket(ctx, bucket)
        require.Error(t, err,"Ожидалась ошибка при создании бакета с дублирующимся ClientID")
        if !errdefs.Is(err, errdefs.ErrConflict) {
            t.Fatalf("Ожидалась ошибка конфликта (ErrConflict), но получена: %v", err)
        }
    })

    t.Run("TokensGreaterThanCapacity", func(t *testing.T) {
        clearTable(t)

        bucket := &models.Bucket{
            ClientID: "overflow-client",
            Capacity: 5,
            Tokens:   10, // tokens > capacity
        }
        // Создание бакета с tokens > capacity должно привести к ошибке
        err := repo.CreateBucket(ctx, bucket)
        require.Error(t, err, "Ожидалась ошибка при создании бакета с tokens > capacity")
        if !errdefs.Is(err, errdefs.ErrInvalidInput) {
            t.Fatalf("Ожидалась ошибка конфликта (ErrInvalidInput), но получена: %v", err)
        }
    })

    t.Run("TryConsume_SuccessAndEmpty", func(t *testing.T) {
        clearTable(t)

        clientID := "consumer-client"
        bucket := &models.Bucket{
            ClientID: clientID,
            Capacity: 3,
            Tokens:   3,
        }

        err := repo.CreateBucket(ctx, bucket)
        require.NoError(t, err)

        // Последовательно потребляем токены
        for i := 0; i < bucket.Capacity; i++ {
            err := repo.TryConsume(ctx, clientID)
            require.NoError(t, err, "Ошибка при потреблении токена")

            got, _ := repo.GetBucket(ctx, clientID)
            expectedTokens := bucket.Capacity - (i + 1)
            require.Equal(t, expectedTokens, got.Tokens, "Неправильное число токенов после TryConsume")
        }

        err = repo.TryConsume(ctx, clientID)
        require.Equal(t, err, errdefs.NotEnoughTokens, "Ожидалась ошибка NotEnoughTokens при TryConsume из пустого бакета")
    })

    t.Run("RefillTokens", func(t *testing.T) {
        clearTable(t)

        clientID := "refill-client"
        initialTokens := 1
        capacity := 5
        bucket := &models.Bucket{
            ClientID: clientID,
            Capacity: capacity,
            Tokens:   initialTokens,
        }

        err := repo.CreateBucket(ctx, bucket)
        require.NoError(t, err)

        // Заполняем токены до capacity
        err = repo.RefillTokens(ctx, clientID)
        require.NoError(t, err, "Ошибка при RefillTokens")

        // ?token == initialTokens+cfg.Bucket.Refill.Amount
        got, err := repo.GetBucket(ctx, clientID)
        require.NoError(t, err)
        require.Equal(t, 
            initialTokens+cfg.Bucket.Refill.Amount,
            got.Tokens,
            "Токены после RefillTokens должны быть равны capacity")
    })
}
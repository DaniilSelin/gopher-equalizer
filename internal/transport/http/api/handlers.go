package api

import (
    "gopher-equalizer/config"
	"gopher-equalizer/internal/models"
	"gopher-equalizer/internal/errdefs"
	"gopher-equalizer/internal/logger"
    "gopher-equalizer/internal/interfaces"

    "fmt"
	"context"
	"encoding/json"
	"net/http"
    "strconv"

	"go.uber.org/zap"
    "github.com/google/uuid"
)

type Handler struct {
	ctx context.Context
    cfg *config.Config
	bsrv interfaces.IBucketService
}

func NewHandler(ctx context.Context, cfg *config.Config, bsrv interfaces.IBucketService) *Handler {
	return &Handler{
		bsrv: bsrv,
		ctx: ctx,
        cfg: cfg,
	}
}

// удобно
func encode[T any](w http.ResponseWriter, r *http.Request, status int, v T) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

// так как frontend фактически нет, то я тут генерирую RequstID
// Подразумевая, что сразу несколько человек могут пользоваться "панелью администратора"
func GenerateRequestID(ctx context.Context) context.Context {
    return context.WithValue(ctx, logger.RequestID, uuid.New().String())
}

// handleCreateBucket обрабатывает POST /clients
func (h *Handler) handleCreateBucket() http.Handler {
    ctx := GenerateRequestID(h.ctx)
    logger := logger.GetLoggerFromCtx(ctx)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.Info(ctx, "incoming request",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        )

        payload, err := decode[models.Bucket](r)
        if err != nil {
            logger.Info(ctx, "invalid JSON payload", zap.Error(err))
            http.Error(w, "Bad Request", http.StatusBadRequest)
            return
        }
        if err := h.bsrv.CreateBucket(ctx, &payload); err != nil {
            handleServiceError(ctx, w, err)
            return
        }

        logger.Info(ctx, "bucket created",
            zap.String("client_id", payload.ClientID),
            zap.Int("capacity", payload.Capacity),
        )

        encode(w, r, http.StatusCreated, payload)
    })
}

// handleListBuckets обрабатывает GET /clients?limit=&offset=
func (h *Handler) handleListBuckets() http.Handler {
    ctx := GenerateRequestID(h.ctx)
    logger := logger.GetLoggerFromCtx(ctx)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.Info(ctx, "incoming request",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        )

        q := r.URL.Query()
        limit := parseInt(q.Get("limit"), h.cfg.API.DefaultLimit)
        offset := parseInt(q.Get("offset"), 0)
        buckets, err := h.bsrv.ListBuckets(ctx, limit, offset)
        if err != nil {
            handleServiceError(ctx, w, err)
            return
        }

        logger.Info(ctx, "listed buckets",
            zap.Int("returned", len(*buckets)),
        )
        encode(w, r, http.StatusOK, buckets)
    })
}

// handleGetBucket обрабатывает GET /clients/{id}
func (h *Handler) handleGetBucket() http.Handler {
    ctx := GenerateRequestID(h.ctx)
    logger := logger.GetLoggerFromCtx(ctx)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.Info(ctx, "incoming request",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        )

        clientID := r.URL.Path[len("/buckets/"):]
        if clientID == "" {
            logger.Info(ctx, "client_id missing")
            http.Error(w, "clientID required", http.StatusBadRequest)
            return
        }
        bucket, err := h.bsrv.GetBucket(ctx, clientID)
        if err != nil {
            handleServiceError(ctx, w, err)
            return
        }

        logger.Info(ctx, "fetched bucket", zap.String("client_id", clientID))
        encode(w, r, http.StatusOK, bucket)
    })
}

// handleUpdateCapacity обрабатывает PUT /clients/{id}
func (h *Handler) handleUpdateCapacity() http.Handler {
    ctx := GenerateRequestID(h.ctx)
    logger := logger.GetLoggerFromCtx(ctx)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.Info(ctx, "incoming request",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        )

        clientID := r.URL.Path[len("/buckets/"):]
        payload, err := decode[struct{ Capacity int `json:"capacity"`}](r)
        if err != nil {
            logger.Info(ctx, "invalid JSON payload", zap.Error(err))
            http.Error(w, "Bad Request", http.StatusBadRequest)
            return
        }
        if err := h.bsrv.UpdateCapacity(ctx, clientID, payload.Capacity); err != nil {
            handleServiceError(ctx, w, err)
            return
        }

        logger.Info(ctx, "capacity updated",
            zap.String("client_id", clientID),
            zap.Int("new_capacity", payload.Capacity),
        )
        w.WriteHeader(http.StatusNoContent)
    })
}

// handleUpdateTokens обрабатывает PATCH /clients/{id}
func (h *Handler) handleUpdateTokens() http.Handler {
    ctx := GenerateRequestID(h.ctx)
    logger := logger.GetLoggerFromCtx(ctx)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.Info(ctx, "incoming request",
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        )

        clientID := r.URL.Path[len("/buckets/"):]
        payload, err := decode[struct{ Tokens int `json:"tokens"`}](r)
        if err != nil {
            logger.Info(ctx, "invalid JSON payload", zap.Error(err))
            http.Error(w, "Bad Request", http.StatusBadRequest)
            return
        }
        if err := h.bsrv.UpdateTokens(ctx, clientID, payload.Tokens); err != nil {
            handleServiceError(ctx, w, err)
            return
        }


        logger.Info(ctx, "tokens updated",
            zap.String("client_id", clientID),
            zap.Int("new_tokens", payload.Tokens),
        )
        w.WriteHeader(http.StatusNoContent)
    })
}

// handleDeleteBucket обрабатывает DELETE /clients/{id}
func (h *Handler) handleDeleteBucket() http.Handler {
    ctx := GenerateRequestID(h.ctx)
    logger := logger.GetLoggerFromCtx(ctx)

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger.Info(ctx, "incoming request",
            zap.String("method", r.Method), 
            zap.String("path", r.URL.Path),
        )

        clientID := r.URL.Path[len("/buckets/"):]
        if err := h.bsrv.RemoveBucket(ctx, clientID); err != nil {
            handleServiceError(ctx, w, err)
            return
        }

        logger.Info(ctx, "bucket deleted", zap.String("client_id", clientID))
        w.WriteHeader(http.StatusNoContent)
    })
}

// возвращает нужную ошибку
// чуть медленее чем на месте (много лишних проверок)
// зато код более компактный и читаемый
func handleServiceError(ctx context.Context, w http.ResponseWriter, err error) {
    switch {
    case errdefs.Is(err, errdefs.ErrNotFound):
        http.Error(w, "Not Found", http.StatusNotFound)
    case errdefs.Is(err, errdefs.ErrInvalidInput):
        http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
    case errdefs.Is(err, errdefs.ErrConflict):
        http.Error(w, "Conflict: "+err.Error(), http.StatusConflict)
    default:
        logger.GetLoggerFromCtx(ctx).Error(ctx, "internal error", zap.Error(err))
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    }
}

func parseInt(s string, def int) int {
    if s == "" {
        return def
    }
    v, err := strconv.Atoi(s)
    if err != nil {
        return def
    }
    return v
}

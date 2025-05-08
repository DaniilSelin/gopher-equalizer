package http

import (
    "context"
    "net"
    "net/http"
    "time"
    "sync"

    "gopher-equalizer/internal/interfaces"
    "gopher-equalizer/internal/logger"
    "gopher-equalizer/config"
    "go.uber.org/zap"
)

type HealthChecker struct {
	mu sync.RWMutex
	cfg *config.Config
	balancer interfaces.IBalancer
}

func NewHealthChecker(cfg *config.Config, balancer interfaces.IBalancer) *HealthChecker {
	return &HealthChecker{
		cfg: cfg,
		balancer: balancer,
	}
}

func (hc HealthChecker) StartHealthChecks(ctx context.Context) {
	interval := time.Duration(hc.cfg.Proxy.HealthChecker.Interval)
	ticker := time.NewTicker(interval)    

	go func() {
        defer ticker.Stop()
        for {
            select {
            	// У нас есть Gracefull Shutdown
            case <-ctx.Done():
                return
            case <-ticker.C:
                hc.runOnce(ctx)
            }
        }
    }()
}

func (hc *HealthChecker) runOnce(ctx context.Context) {
	logger := logger.GetLoggerFromCtx(ctx)

	alive := make(
		[]string,
		0,
		len(hc.cfg.Balancer.Backends),
	)

    for _, addr := range hc.cfg.Balancer.Backends {
        if checkOne(
        	addr,
        	time.Duration(hc.cfg.Proxy.HealthChecker.HealthCheckTimeout),
        ) {
            alive = append(alive, addr)
        } else {
            logger.Info(ctx, "HEALTH-CHECK: failed", zap.String("backend", addr))
        }
    }

    logger.Info(ctx, "HEALTH-CHECK: result",
        zap.Int("alive", len(alive)),
        zap.Int("total", len(hc.cfg.Balancer.Backends)),
    )

  	hc.balancer.ResetBackends(alive)
}

func checkOne(addr string, timeout time.Duration) bool {
    client := &http.Client{Timeout: timeout}
    resp, err := client.Get(addr)
    if err == nil {
        resp.Body.Close()
        return resp.StatusCode < 500
    }
    // Если HTTP не прошёл, но хост доступен по TCP
    // то мы всё равно считаем «alive», так как ответ пришел не с 500-кой
    conn, err2 := net.DialTimeout("tcp", addr[len("http://"):], timeout)
    if err2 == nil {
        conn.Close()
        return true
    }
    return false
}

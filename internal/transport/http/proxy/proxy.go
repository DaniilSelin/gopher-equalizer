package proxy

import (
    "context"
    "net"
    "net/http"
    "net/http/httputil"
    "net/url"
    "time"

    "gopher-equalizer/internal/balancer"
    "gopher-equalizer/internal/logger"
    "gopher-equalizer/internal/errdefs"
    "gopher-equalizer/config"
    "gopher-equalizer/internal/interfaces"

    "go.uber.org/zap"
    "github.com/google/uuid"
)

const proxyErrorKey = "proxyErr"

type Proxy struct {
    rp        *httputil.ReverseProxy
    balancer  *balancer.Balancer
    bsrv interfaces.IBucketService
    cfg *config.Config
    logger *logger.Logger
}

func NewProxy(cfg *config.Config, bal *balancer.Balancer, bsrv interfaces.IBucketService, logger *logger.Logger) *Proxy {
    // Вынести в конфиг
    transport := &http.Transport{
        Proxy: http.ProxyFromEnvironment,
        DialContext: (&net.Dialer{
            Timeout:   time.Duration(cfg.Proxy.Timeout),
            KeepAlive: time.Duration(cfg.Proxy.KeepAlive),
        }).DialContext,
        IdleConnTimeout:     time.Duration(cfg.Proxy.IdleConnTimeout),
        MaxIdleConns:        cfg.Proxy.MaxIdleConns,
        MaxIdleConnsPerHost: cfg.Proxy.MaxIdleConnsPerHost,
        TLSHandshakeTimeout: time.Duration(cfg.Proxy.TLSHandshakeTimeout),
    }

    p := &Proxy{
        balancer:     bal,
        bsrv:    bsrv,
        logger:  logger,
    }

    p.rp = &httputil.ReverseProxy{
        Director:     p.director,
        Transport:    transport,
        ErrorHandler: p.errHandler,
    }

    return p
}

func (p *Proxy) errHandler(w http.ResponseWriter, req *http.Request, err error) {
    ctx := req.Context()

    if ctxErr, ok := req.Context().Value(proxyErrorKey).(error); ok {
        if errdefs.Is(ctxErr, errdefs.ErrNoBackends) {
            p.logger.Info(ctx, "no backends")
            http.Error(w, "Service Unavailable: no healthy backends", http.StatusServiceUnavailable)
            return
        }
        if errdefs.Is(ctxErr, errdefs.ErrRateLimitExceeded) {
            p.logger.Info(ctx, "rate limit")
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        p.logger.Error(ctx, "proxy error", zap.Error(ctxErr))
        // Неизвестная ошибка — можно вернуть 502
        http.Error(w, "Bad Gateway", http.StatusBadGateway)
        return
    }

    p.logger.Error(ctx, "unexpected proxy error")
    // Если ошибка не из контекста — стандартная обработка
    http.Error(w, "Proxy error", http.StatusBadGateway)
}

func (p *Proxy) director(req *http.Request) {
    ctx := GenerateRequestID(req.Context())
    req = req.WithContext(ctx)
    ctx = logger.SetLoggerInCtx(ctx, p.logger)

    ip, _, err := net.SplitHostPort(req.RemoteAddr)
    if err != nil {
        p.logger.Info(ctx, "invalid client address", zap.Error(err))
        ip = req.RemoteAddr
    }

    if err := p.bsrv.TryConsume(ctx, ip); err != nil {
        p.logger.Info(ctx, "rate limit exceeded", zap.String("client_ip", ip), zap.Error(err))
        req = req.WithContext(context.WithValue(ctx, proxyErrorKey, err))
        return
    }

    backend, err := p.balancer.NextBackend()
    if err != nil {
        p.logger.Info(ctx, "no available backends", zap.Error(err))
        req = req.WithContext(context.WithValue(ctx, proxyErrorKey, err))
        return
    }

    target, _ := url.Parse(backend)
    req.URL.Scheme = target.Scheme
    req.URL.Host = target.Host
    req.Host = target.Host

    p.logger.Info(ctx, "proxying request",
        zap.String("backend", backend),
        zap.String("path", req.URL.Path),
    )
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    backend, err := p.balancer.NextBackend()
    if err != nil {
        p.logger.Info(ctx, "no backends available", zap.Error(err))
        http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
        return
    }
    // Записываем в лог, на какой backend уйдёт запрос
    p.logger.Info(ctx, "proxy to backend",
        zap.String("backend", backend),
        zap.String("method", r.Method),
        zap.String("path", r.URL.Path),
    )
    // Меняем URL внутри запроса и передаём в ReverseProxy
    // (Director внутри rp сработает заново, но host уже выбран)
    p.rp.ServeHTTP(w, r)
}

func GenerateRequestID(ctx context.Context) context.Context {
    return context.WithValue(ctx, logger.RequestID, uuid.New().String())
}
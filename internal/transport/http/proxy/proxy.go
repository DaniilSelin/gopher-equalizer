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

type errorTransport struct {
    base http.RoundTripper
}

func NewProxy(cfg *config.Config, bal *balancer.Balancer, bsrv interfaces.IBucketService, logger *logger.Logger) *Proxy {
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
    if errdefs.Is(err, errdefs.ErrNoBackends) {
        p.logger.Info(req.Context(), "no backends")
        http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
        return
    }

    p.logger.Error(req.Context(), "PROXY: unexpected proxy error")
    http.Error(w, "Bad Gateway", http.StatusBadGateway)
}

func (p *Proxy) director(req *http.Request) {
    ctx := GenerateRequestID(req.Context())
    req = req.WithContext(ctx)
    ctx = logger.SetLoggerInCtx(ctx, p.logger)

    backend, err := p.balancer.NextBackend()
    if err != nil {
        p.logger.Info(ctx, "PROXY-DIRECTION: no available backends", zap.Error(err))
        req = req.WithContext(context.WithValue(ctx, proxyErrorKey, err))
        return
    }

    target, _ := url.Parse(backend)
    req.URL.Scheme = target.Scheme
    req.URL.Host = target.Host
    req.Host = target.Host

    p.logger.Info(ctx, "PROXY-DIRECTION: proxying request",
        zap.String("backend", backend),
        zap.String("path", req.URL.Path),
    )
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := logger.SetLoggerInCtx(r.Context(), p.logger)
    ctx = GenerateRequestID(ctx)
    ip, _, _ := net.SplitHostPort(r.RemoteAddr)

    if err := p.bsrv.TryConsume(ctx, ip); err != nil {
        p.logger.Info(ctx, "rate limit exceeded", zap.String("client_ip", ip), zap.Error(err))
        http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
        return
    }

    // backend, err := p.balancer.NextBackend()
    // if err != nil {
    //     p.logger.Info(ctx, "no backends available", zap.Error(err))
    //     http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
    //     return
    // }

    p.logger.Info(ctx, "proxy to backend",
        // zap.String("backend", backend),
        zap.String("method", r.Method),
        zap.String("path", r.URL.Path),
    )

    p.rp.ServeHTTP(w, r)
}

func GenerateRequestID(ctx context.Context) context.Context {
    return context.WithValue(ctx, logger.RequestID, uuid.New().String())
}
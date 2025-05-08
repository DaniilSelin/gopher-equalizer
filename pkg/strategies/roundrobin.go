package strategies

import (
	"fmt"
	"sync"
)

type RoundRobin struct {
    servers []string
    index   int
    mu      sync.RWMutex
}

func NewRoundRobin(servers []string) *RoundRobin {
    return &RoundRobin{servers: servers}
}

func (rr *RoundRobin) Next() (string, error) {
    rr.mu.Lock()
    defer rr.mu.Unlock()
    if len(rr.servers) == 0 {
        return "", fmt.Errorf("no backends available")
    }
    server := rr.servers[rr.index]
    rr.index = (rr.index + 1) % len(rr.servers)
    return server, nil
}

func (rr *RoundRobin) ResetBackends(backs []string) {
    rr.mu.Lock()
    defer rr.mu.Unlock()
    rr.servers = backs
}
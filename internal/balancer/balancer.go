package balancer

import (
    "gopher-equalizer/internal/interfaces"
)

type Balancer struct {
    strat interfaces.IStrategy
}

func NewBalancer(strategy interfaces.IStrategy) *Balancer {
    return &Balancer{
        strat: strategy,
    }
}

func (b *Balancer) NextBackend() (string, error) {
    return b.strat.Next()
}

func (b *Balancer) ResetBackends(backs []string) {
    b.strat.ResetBackends(backs)
}
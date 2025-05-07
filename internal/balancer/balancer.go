package balancer

import (
    "gopher-equalizer/internal/interfaces"
)

type Balancer struct {
    strat interfaces.Strategy
}

func NewBalancer(strat interfaces.Strategy) *Balancer {
    return &Balancer{strat: strat}
}

func (b *Balancer) NextBackend() (string, error) {
    return b.strat.Next()
}
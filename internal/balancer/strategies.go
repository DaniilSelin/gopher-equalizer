package balancer

import (
    "fmt"
    "gopher-equalizer/pkg/strategies"
    "gopher-equalizer/internal/interfaces"
)

var strategyFactories = map[string]func([]string) interfaces.IStrategy{
    "round_robin": func(backends []string) interfaces.IStrategy {
        return strategies.NewRoundRobin(backends)
    },
    // "random": func(backends []string) interfaces.IStrategy {
    //     return strategies.NewRandom(backends)
    // },
}

func CreateStrategy(name string, backends []string) (interfaces.IStrategy, error) {
    if factory, ok := strategyFactories[name]; ok {
        return factory(backends), nil
    }
    return nil, fmt.Errorf("unknown strategy: %s", name)
}
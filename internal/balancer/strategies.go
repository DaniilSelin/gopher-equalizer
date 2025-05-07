package balancer

import (
    "fmt"
    "gopher-equalizer/pkg/strategies"
    "gopher-equalizer/internal/interfaces"
)

var strategyFactories = map[string]func([]string) interfaces.Strategy{
    "round_robin": func(backends []string) interfaces.Strategy {
        return strategies.NewRoundRobin(backends)
    },
    // "random": func(backends []string) interfaces.Strategy {
    //     return strategies.NewRandom(backends)
    // },
}

func CreateStrategy(name string, backends []string) (interfaces.Strategy, error) {
    if factory, ok := strategyFactories[name]; ok {
        return factory(backends), nil
    }
    return nil, fmt.Errorf("unknown strategy: %s", name)
}
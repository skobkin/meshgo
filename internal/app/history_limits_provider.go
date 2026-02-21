package app

import "github.com/skobkin/meshgo/internal/config"

type historyLimitsProvider struct {
	currentConfig func() config.AppConfig
}

func newHistoryLimitsProvider(currentConfig func() config.AppConfig) historyLimitsProvider {
	return historyLimitsProvider{currentConfig: currentConfig}
}

func (p historyLimitsProvider) PositionHistoryLimit() int {
	return p.limitOrDefault(
		func(cfg config.AppConfig) *int { return cfg.Persistence.HistoryLimits.Position },
		config.DefaultPositionHistoryLimit,
	)
}

func (p historyLimitsProvider) TelemetryHistoryLimit() int {
	return p.limitOrDefault(
		func(cfg config.AppConfig) *int { return cfg.Persistence.HistoryLimits.Telemetry },
		config.DefaultTelemetryHistoryLimit,
	)
}

func (p historyLimitsProvider) IdentityHistoryLimit() int {
	return p.limitOrDefault(
		func(cfg config.AppConfig) *int { return cfg.Persistence.HistoryLimits.Identity },
		config.DefaultIdentityHistoryLimit,
	)
}

func (p historyLimitsProvider) limitOrDefault(selectLimit func(config.AppConfig) *int, fallback int) int {
	if p.currentConfig == nil {
		return fallback
	}
	cfg := p.currentConfig()
	value := selectLimit(cfg)
	if value == nil {
		return fallback
	}

	return *value
}

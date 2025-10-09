package config

import (
	"context"
)

type ctxKey string

var configKey ctxKey = "tfsnap_config"

func ToContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

func FromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configKey).(*Config); ok {
		return cfg
	}
	return nil
}

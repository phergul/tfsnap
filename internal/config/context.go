package config

import (
	"context"
)

func (c *Config) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, c, c)
}

func (c *Config) FromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(c).(*Config); ok {
		return cfg
	}
	return nil
}

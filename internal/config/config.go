package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Route struct {
	Prefix   string `json:"prefix"`
	Upstream string `json:"upstream"`
}

type Config struct {
	ListenAddr string  `json:"listen_addr"`
	Routes     []Route `json:"routes"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.ListenAddr == "" {
		return errors.New("listen_addr is required")
	}
	if len(c.Routes) == 0 {
		return errors.New("at least one route is required")
	}

	seen := make(map[string]bool)
	for i, rt := range c.Routes {
		if !strings.HasPrefix(rt.Prefix, "/") {
			return fmt.Errorf("route %d (%q): prefix must start with /", i, rt.Prefix)
		}
		if strings.HasSuffix(rt.Prefix, "/") {
			return fmt.Errorf("route %d (%q): prefix must not end with /", i, rt.Prefix)
		}
		if seen[rt.Prefix] {
			return fmt.Errorf("route %d (%q): duplicate prefix", i, rt.Prefix)
		}
		seen[rt.Prefix] = true

		u, err := url.Parse(rt.Upstream)
		if err != nil {
			return fmt.Errorf("route %d (%q): bad upstream: %w", i, rt.Prefix, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("route %d (%q): upstream %q must use http or https", i, rt.Prefix, rt.Upstream)
		}
		if u.Host == "" {
			return fmt.Errorf("route %d (%q): upstream %q has no host", i, rt.Prefix, rt.Upstream)
		}
	}
	return nil
}

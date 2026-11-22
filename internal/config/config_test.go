package config

import "testing"

func TestValidate(t *testing.T) {
	// valid returns a fresh, known-good config. Each test case modifies its own copy.
	valid := func() Config {
		return Config{
			ListenAddr: ":8080",
			Routes: []Route{
				{Prefix: "/orders", Upstream: "http://localhost:8081"},
			},
		}
	}

	tests := []struct {
		name    string
		modify  func(c *Config)
		wantErr bool
	}{
		{"valid config", func(c *Config) {}, false},
		{"https upstream is allowed", func(c *Config) { c.Routes[0].Upstream = "https://example.com" }, false},
		{"missing listen addr", func(c *Config) { c.ListenAddr = "" }, true},
		{"no routes", func(c *Config) { c.Routes = nil }, true},
		{"prefix without leading slash", func(c *Config) { c.Routes[0].Prefix = "orders" }, true},
		{"prefix with trailing slash", func(c *Config) { c.Routes[0].Prefix = "/orders/" }, true},
		{"duplicate prefix", func(c *Config) { c.Routes = append(c.Routes, c.Routes[0]) }, true},
		{"upstream without scheme", func(c *Config) { c.Routes[0].Upstream = "localhost:8081" }, true},
		{"upstream with ftp scheme", func(c *Config) { c.Routes[0].Upstream = "ftp://localhost:21" }, true},
		{"upstream without host", func(c *Config) { c.Routes[0].Upstream = "http://" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid()
			tt.modify(&cfg)

			err := cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

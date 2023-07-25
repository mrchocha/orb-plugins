package etcd

import (
	"crypto/tls"
	"fmt"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source/cli"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"go.uber.org/zap"
)

// Name provides the name of this registry.
const Name = "etcd"

const (
	DefaultAddresses = "asdsada" // TODO:  ask what should be here
)

func init() {
	//nolint:errcheck
	_ = cli.Flags.Add(cli.NewFlag(
		"registry_addresses",
		DefaultAddresses, // TODO:  ask what should be here
		cli.ConfigPathSlice([]string{"registry", "addresses"}),
		cli.Usage("Registry addresses."),
	))

	if err := registry.Plugins.Add(Name, registry.ProviderFunc(ProvideRegistryEtcd)); err != nil {
		panic(err)
	}
}

type authCreds struct {
	Username string
	Password string
}

// Config provides configuration for the etcd registry.
type Config struct {
	registry.Config `yaml:",inline"`

	Addresses []string    `json:"addresses,omitempty" yaml:"addresses,omitempty"`
	Secure    bool        `json:"secure,omitempty" yaml:"secure,omitempty"`
	TLSConfig *tls.Config `json:"-" yaml:"-"`

	Auth      *authCreds  `json:"auth,omitempty" yaml:"auth,omitempty"`
	LogConfig *zap.Config `json:"logConfig,omitempty" yaml:"logConfig,omitempty"`
}

// NewConfig creates a new config object.
func NewConfig(
	serviceName types.ServiceName,
	datas types.ConfigData,
	opts ...registry.Option,
) (Config, error) {
	cfg := Config{
		Config: registry.NewConfig(),
	}

	cfg.ApplyOptions(opts...)

	sections := types.SplitServiceName(serviceName)
	if err := config.Parse(append(sections, registry.ComponentType), datas, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func WithAddress(n ...string) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Addresses = n
		} else {
			panic(fmt.Sprintf("wrong type: %T", c))
		}
	}
}

// WithSecure defines if we want a secure connection to nats.
func WithSecure(n bool) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Secure = n
		}
	}
}

// WithTLSConfig defines the TLS config to use for the secure connection.
func WithTLSConfig(n *tls.Config) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.TLSConfig = n
		}
	}
}

// WithLogConfig sets the logConfigs.
func WithLogConfig(config *zap.Config) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.LogConfig = config
		}
	}
}

// WithAuth sets the auth creds.
func WithAuth(username, password string) registry.Option {
	return func(c registry.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Auth = &authCreds{Username: username, Password: password}
		}
	}
}

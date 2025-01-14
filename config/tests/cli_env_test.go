package test

import (
	"net/url"
	"os"
	"testing"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIEnv(t *testing.T) {
	// Clear flags from other tests
	cli.Flags.Clear()

	err := cli.Flags.Add(cli.NewFlag(
		"registry",
		"mdns",
		cli.EnvVars("ORB_REGISTRY"),
		cli.ConfigPath("registry.plugin"),
		cli.Usage("string flag usage"),
	))
	require.NoError(t, err)

	err = cli.Flags.Add(cli.NewFlag(
		"registry_ttl",
		300,
		cli.EnvVars("ORB_REGISTRY_TTL"),
		cli.ConfigPathSlice([]string{"registry", "ttl"}),
		cli.Usage("int flag usage"),
	))
	require.NoError(t, err)

	err = cli.Flags.Add(cli.NewFlag(
		"nats-address",
		[]string{},
		cli.EnvVars("ORB_REGISTRY_NATS_ADDRESS"),
		cli.ConfigPathSlice([]string{"registry", "addresses"}),
		cli.Usage("NATS Address"),
	))
	require.NoError(t, err)

	// Set Environ variables.
	require.NoError(t, os.Setenv("ORB_REGISTRY", "nats"))
	require.NoError(t, os.Setenv("ORB_REGISTRY_TTL", "600"))
	require.NoError(t, os.Setenv("ORB_REGISTRY_NATS_ADDRESS", "nats://localhost:4222"))

	os.Args = []string{
		"testapp",
	}

	u1, err := url.Parse("cli://urfave")
	require.NoError(t, err)

	datas, err := config.Read([]*url.URL{u1}, []string{})
	require.NoError(t, err)

	// Merge all data from the URL's.
	cfg := newRegistryNatsConfig()
	err = config.Parse([]string{"registry"}, datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.Equal(t, true, cfg.Enabled, "Enabled by default")
	assert.Equal(t, "nats", cfg.Plugin, "Plugin")
	assert.Equal(t, 600, cfg.Timeout, "Timeout")
	assert.Equal(t, true, cfg.Secure, "Secure by default")
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses, "Addresses")
}

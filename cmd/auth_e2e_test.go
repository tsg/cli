package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/kr/pty"
	"github.com/stretchr/testify/require"
	"github.com/xata/cli/internal/envcfg"
)

type TestConfig struct {
	TestKey string `env:"TEST_API_KEY`
}

func GetTestConfigFromEnv() (config TestConfig, err error) {
	err = envcfg.ReadEnv([]string{"../.env", "../.env.local"})
	if err != nil {
		return TestConfig{}, err
	}

	config.TestKey = os.Getenv("TEST_API_KEY")
	return
}

func TestAuthStatusNotLoggedInCommand(t *testing.T) {
	c, err := expect.NewConsole(
		expect.WithDefaultTimeout(1 * time.Second),
	)
	require.NoError(t, err)
	defer c.Close()

	configDir, err := ioutil.TempDir("", "xata-config")
	require.NoError(t, err)
	defer os.RemoveAll(configDir)

	cmd := exec.Command(
		"../xata",
		fmt.Sprintf("--configdir=%s", configDir),
		"auth", "status")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	require.NoError(t, err)

	_, err = c.ExpectString("You are not logged in, run `xata auth login` first")
	require.NoError(t, err)

	cmd.Wait()
	require.NoError(t, err)
}

func TestAuthLoginCommand(t *testing.T) {
	config, err := GetTestConfigFromEnv()
	require.NoError(t, err)
	require.NotEmpty(t, config.TestKey)

	c, err := NewVT10XConsole(
		expect.WithDefaultTimeout(1 * time.Second),
	)
	require.NoError(t, err)
	defer c.Close()

	configDir, err := ioutil.TempDir("", "xata-config")
	require.NoError(t, err)
	defer os.RemoveAll(configDir)

	tests := []struct {
		name      string
		apiKey    string
		procedure func(t *testing.T, c *expect.Console, apiKey string)
	}{
		{
			name:   "login with valid API key",
			apiKey: config.TestKey,
			procedure: func(t *testing.T, c *expect.Console, apiKey string) {
				_, err = c.ExpectString("Introduce your API key:")
				require.NoError(t, err)
				_, err = c.SendLine(config.TestKey)
				require.NoError(t, err)

				_, err = c.Expect(
					expect.String("Checking access to the API...OK"),
					expect.WithTimeout(5*time.Second))
				require.NoError(t, err)

				_, err = c.ExpectString("All set! you can now start using xata")
				require.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run("test", func(t *testing.T) {
			cmd := exec.Command(
				"../xata",
				fmt.Sprintf("--configdir=%s", configDir),
				"auth", "login")
			cmd.Stdin = c.Tty()
			cmd.Stdout = c.Tty()
			cmd.Stderr = c.Tty()

			err = cmd.Start()
			require.NoError(t, err)

			test.procedure(t, c, test.apiKey)

			err = c.Close()
			require.NoError(t, err)
			cmd.Wait()
			require.NoError(t, err)
		})
	}
}

// NewVT10XConsole returns a new expect.Console that multiplexes the
// Stdin/Stdout to a VT10X terminal, allowing Console to interact with an
// application sending ANSI escape sequences.
func NewVT10XConsole(opts ...expect.ConsoleOpt) (*expect.Console, error) {
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, err
	}

	term := vt10x.New(vt10x.WithWriter(pts))

	c, err := expect.NewConsole(append(opts, expect.WithStdin(ptm), expect.WithStdout(term), expect.WithCloser(pts, ptm))...)
	if err != nil {
		return nil, err
	}

	return c, nil
}

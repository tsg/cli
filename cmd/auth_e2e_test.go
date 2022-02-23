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

const e2eXataCommand = "../xata"

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

func TestAuthLoginCommand(t *testing.T) {
	config, err := GetTestConfigFromEnv()
	require.NoError(t, err)
	require.NotEmpty(t, config.TestKey)

	configDir, err := ioutil.TempDir("", "xata-config")
	require.NoError(t, err)
	defer os.RemoveAll(configDir)

	tests := []struct {
		name      string
		procedure func(t *testing.T, c *expect.Console)
		exitCode  int
	}{
		{
			name: "try an invalid API key",
			procedure: func(t *testing.T, c *expect.Console) {
				_, err = c.ExpectString("Introduce your API key:")
				require.NoError(t, err)
				_, err = c.SendLine("invalid_key")
				require.NoError(t, err)

				_, err = c.Expect(
					expect.String("Checking access to the API...Auth error: Invalid API key"),
					expect.WithTimeout(5*time.Second))
				require.NoError(t, err)

				_, err = c.ExpectString("For more information please see https://docs.xata.io/cli/getting-started")
				require.NoError(t, err)
			},
			exitCode: 1,
		},
		{
			name: "login with valid API key",
			procedure: func(t *testing.T, c *expect.Console) {
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
			exitCode: 0,
		},
		{
			name: "login again, should ask for a confirmation",
			procedure: func(t *testing.T, c *expect.Console) {
				_, err = c.ExpectString("Authentication is already configured, do you want to override it?")
				require.NoError(t, err)
				_, err = c.SendLine("y")
				require.NoError(t, err)

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
			exitCode: 0,
		},
		{
			name: "answer No this time",
			procedure: func(t *testing.T, c *expect.Console) {
				_, err = c.ExpectString("Authentication is already configured, do you want to override it?")
				require.NoError(t, err)
				_, err = c.SendLine("N")
				require.NoError(t, err)

				_, err = c.ExpectString("No")
				require.NoError(t, err)
			},
			exitCode: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewVT10XConsole(
				expect.WithDefaultTimeout(1 * time.Second),
				//expect.WithStdout(os.Stdout),
			)
			require.NoError(t, err)
			defer c.Close()

			cmd := exec.Command(
				e2eXataCommand,
				fmt.Sprintf("--configdir=%s", configDir),
				"auth", "login")
			cmd.Stdin = c.Tty()
			cmd.Stdout = c.Tty()
			cmd.Stderr = c.Tty()

			err = cmd.Start()
			require.NoError(t, err)

			test.procedure(t, c)

			err = c.Close()
			require.NoError(t, err)
			err = cmd.Wait()
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					require.NoError(t, err)
				}
			}
			require.Equal(t, test.exitCode, exitCode)
		})
	}
}

func startCommand(t *testing.T, configDir string, args ...string) (*expect.Console, *exec.Cmd) {
	c, err := NewVT10XConsole(
		expect.WithDefaultTimeout(1 * time.Second),
		//expect.WithStdout(os.Stdout),
	)
	require.NoError(t, err)

	args = append([]string{
		fmt.Sprintf("--configdir=%s", configDir),
	}, args...)

	cmd := exec.Command(e2eXataCommand, args...)
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	require.NoError(t, err)

	return c, cmd
}

func loginWithKey(t *testing.T, configDir string, apiKey string) {
	c, cmd := startCommand(t, configDir, "auth", "login")
	defer c.Close()

	_, err := c.ExpectString("Introduce your API key:")
	require.NoError(t, err)
	_, err = c.SendLine(apiKey)
	require.NoError(t, err)

	_, err = c.Expect(
		expect.String("Checking access to the API...OK"),
		expect.WithTimeout(5*time.Second))
	require.NoError(t, err)

	_, err = c.ExpectString("All set! you can now start using xata")
	require.NoError(t, err)

	err = cmd.Wait()
	require.NoError(t, err)
}

func TestAuthStatus(t *testing.T) {
	config, err := GetTestConfigFromEnv()
	require.NoError(t, err)
	require.NotEmpty(t, config.TestKey)

	configDir, err := ioutil.TempDir("", "xata-config")
	require.NoError(t, err)
	defer os.RemoveAll(configDir)

	c, cmd := startCommand(t, configDir, "auth", "status")

	_, err = c.ExpectString("You are not logged in, run `xata auth login` first")
	require.NoError(t, err)

	c.Close()
	cmd.Wait()

	loginWithKey(t, configDir, config.TestKey)

	c, cmd = startCommand(t, configDir, "auth", "status")
	defer c.Close()
	_, err = c.ExpectString("Client is logged in")
	require.NoError(t, err)

	_, err = c.ExpectString("Checking access to the API...OK")
	require.NoError(t, err)

	err = cmd.Wait()
	require.NoError(t, err)

}

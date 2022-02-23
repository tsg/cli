package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func createWorkspaceCommand(t *testing.T, configDir, apiKey, name string) string {
	c, cmd := startCommand(t, configDir, "--nocolor", "workspaces", "create", name)
	defer c.Close()

	output, err := c.ExpectString("}")
	require.NoError(t, err)
	require.NotEmpty(t, output)

	var response struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err = json.Unmarshal([]byte(output), &response)
	require.NoError(t, err)

	require.Equal(t, name, response.Name)
	require.NotEmpty(t, response.ID)

	err = cmd.Wait()
	require.NoError(t, err)

	return response.ID
}

func deleteWorkspaceCommand(t *testing.T, configDir, apiKey, workspaceID string) {
	c, cmd := startCommand(t, configDir, "workspaces", "delete", workspaceID)
	defer c.Close()

	err := cmd.Wait()
	require.NoError(t, err)
}

func TestWorkspacesCreate(t *testing.T) {
	config, err := GetTestConfigFromEnv()
	require.NoError(t, err)
	require.NotEmpty(t, config.TestKey)

	configDir, err := ioutil.TempDir("", "xata-config")
	require.NoError(t, err)
	defer os.RemoveAll(configDir)

	loginWithKey(t, configDir, config.TestKey)

	workspaceID := createWorkspaceCommand(t, configDir, config.TestKey, "test")
	defer deleteWorkspaceCommand(t, configDir, config.TestKey, workspaceID)
}

func TestWorkspacesDeleteNotExistant(t *testing.T) {
	config, err := GetTestConfigFromEnv()
	require.NoError(t, err)
	require.NotEmpty(t, config.TestKey)

	configDir, err := ioutil.TempDir("", "xata-config")
	require.NoError(t, err)
	defer os.RemoveAll(configDir)

	loginWithKey(t, configDir, config.TestKey)

	c, cmd := startCommand(t, configDir, "workspaces", "delete", "test")
	defer c.Close()

	_, err = c.ExpectString("Auth error: no access to the workspace")
	require.NoError(t, err)
	_, err = c.ExpectString("For more information please see https://docs.xata.io/cli/getting-started")
	require.NoError(t, err)

	err = cmd.Wait()
	require.Error(t, err)
}

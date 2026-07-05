package kamal

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeExecCommand(exitCode int, stdout string) func(string, ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("GO_HELPER_EXIT_CODE=%d", exitCode),
			fmt.Sprintf("GO_HELPER_STDOUT=%s", stdout),
		)
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	exitCode := 0
	if code := os.Getenv("GO_HELPER_EXIT_CODE"); code != "" {
		fmt.Sscanf(code, "%d", &exitCode)
	}
	stdout := os.Getenv("GO_HELPER_STDOUT")
	if stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}
	os.Exit(exitCode)
}

func TestManager_Register_Success(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(0, "Service web deployed to 10.0.0.1:8080")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Register("web", "web1", "10.0.0.1", 8080, 100)
	assert.NoError(t, err)
}

func TestManager_Register_Failure(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(1, "deploy failed")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Register("web", "web1", "10.0.0.1", 8080, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kamal-proxy deploy failed")
}

func TestManager_Remove_Success(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(0, "Service web removed")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Remove("web", "web1")
	assert.NoError(t, err)
}

func TestManager_Remove_Failure(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(1, "remove failed")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Remove("web", "web1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kamal-proxy remove failed")
}

func TestManager_Shift_Success(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(0, "Service web deployed")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Shift("web", "10.0.0.2:8080", 50)
	assert.NoError(t, err)
}

func TestManager_Drain_Success(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(0, "Service web paused")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Drain("web", "web1", 30)
	assert.NoError(t, err)
}

func TestManager_Drain_Failure(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()
	ExecCommand = fakeExecCommand(1, "pause failed")

	mgr := New("http://127.0.0.1:80")
	err := mgr.Drain("web", "web1", 30)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kamal-proxy pause failed")
}

func TestManager_Health_Success(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()

	listOutput := `[{"service":"web","host":"web1","target":"10.0.0.1:8080","state":"running","tls":false,"active_connections":5}]`
	ExecCommand = fakeExecCommand(0, listOutput)

	mgr := New("http://127.0.0.1:80")
	result, err := mgr.Health("web", "web1")
	require.NoError(t, err)
	assert.Equal(t, "up", result.Status)
	assert.Equal(t, 100, result.Weight)
	assert.Equal(t, 5, result.Connections)
}

func TestManager_Health_ServiceNotFound(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()

	listOutput := `[{"service":"api","host":"api1","target":"10.0.0.2:9090","state":"running","tls":false,"active_connections":0}]`
	ExecCommand = fakeExecCommand(0, listOutput)

	mgr := New("http://127.0.0.1:80")
	_, err := mgr.Health("web", "web1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestManager_Health_Draining(t *testing.T) {
	origExecCommand := ExecCommand
	defer func() { ExecCommand = origExecCommand }()

	listOutput := `[{"service":"web","host":"web1","target":"10.0.0.1:8080","state":"draining","tls":false,"active_connections":2}]`
	ExecCommand = fakeExecCommand(0, listOutput)

	mgr := New("http://127.0.0.1:80")
	result, err := mgr.Health("web", "web1")
	require.NoError(t, err)
	assert.Equal(t, "drain", result.Status)
	assert.Equal(t, 2, result.Connections)
}

func TestParseServiceHealth_Success(t *testing.T) {
	output := []byte(`[{"service":"web","host":"web1","target":"10.0.0.1:8080","state":"running","tls":false,"active_connections":3}]`)
	result, err := parseServiceHealth(output, "web", "web1")
	require.NoError(t, err)
	assert.Equal(t, "up", result.Status)
	assert.Equal(t, 3, result.Connections)
}

func TestParseServiceHealth_NotFound(t *testing.T) {
	output := []byte(`[{"service":"api","host":"api1","target":"10.0.0.2:9090","state":"running","tls":false,"active_connections":0}]`)
	_, err := parseServiceHealth(output, "web", "web1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestParseServiceHealth_InvalidJSON(t *testing.T) {
	_, err := parseServiceHealth([]byte("not json"), "web", "web1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestParseServiceHealth_PausedState(t *testing.T) {
	output := []byte(`[{"service":"web","host":"web1","target":"10.0.0.1:8080","state":"paused","tls":false,"active_connections":0}]`)
	result, err := parseServiceHealth(output, "web", "web1")
	require.NoError(t, err)
	assert.Equal(t, "drain", result.Status)
}

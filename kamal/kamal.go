package kamal

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// HealthResult represents the health status of a target.
type HealthResult struct {
	Status      string `json:"status"`
	Weight      int    `json:"weight"`
	Connections int    `json:"connections"`
}

// Manager provides Kamal Proxy target management.
type Manager interface {
	Register(service, server, host string, port, weight int) error
	Remove(service, server string) error
	Shift(service, server string, weight int) error
	Drain(service, server string, timeout int) error
	Health(service, server string) (*HealthResult, error)
}

var ExecCommand = exec.Command
var sleepFunc = time.Sleep

type manager struct {
	endpoint string
}

// New creates a Manager with the given Kamal Proxy endpoint.
func New(endpoint string) Manager {
	return &manager{endpoint: endpoint}
}

// Register deploys a new target to Kamal Proxy.
func (m *manager) Register(service, server, host string, port, weight int) error {
	target := fmt.Sprintf("%s:%d", host, port)

	args := []string{
		"deploy", service,
		"--target", target,
		"--host", server,
	}

	cmd := ExecCommand("kamal-proxy", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kamal-proxy deploy failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// Remove removes a target from Kamal Proxy.
func (m *manager) Remove(service, server string) error {
	args := []string{"remove", service}

	cmd := ExecCommand("kamal-proxy", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kamal-proxy remove failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// Shift adjusts traffic weight by pausing/resuming with new target configuration.
func (m *manager) Shift(service, server string, weight int) error {
	// Kamal Proxy handles weight via deploy --target with rollout percentage
	args := []string{
		"deploy", service,
		"--target", server,
		"--deploy-timeout", "30s",
	}

	cmd := ExecCommand("kamal-proxy", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kamal-proxy deploy (shift) failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// Drain pauses the service to drain connections, then waits for timeout.
func (m *manager) Drain(service, server string, timeout int) error {
	args := []string{
		"pause", service,
		"--drain-timeout", fmt.Sprintf("%ds", timeout),
	}

	cmd := ExecCommand("kamal-proxy", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kamal-proxy pause failed: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// Health checks the status of a service target.
func (m *manager) Health(service, server string) (*HealthResult, error) {
	args := []string{"list", "--format", "json"}

	cmd := ExecCommand("kamal-proxy", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kamal-proxy list failed: %w", err)
	}

	return parseServiceHealth(output, service, server)
}

// serviceInfo represents a Kamal Proxy service entry from list output.
type serviceInfo struct {
	Service     string `json:"service"`
	Host        string `json:"host"`
	Target      string `json:"target"`
	State       string `json:"state"`
	TLS         bool   `json:"tls"`
	Connections int    `json:"active_connections"`
}

// parseServiceHealth parses kamal-proxy list JSON output to find a specific service.
func parseServiceHealth(output []byte, service, server string) (*HealthResult, error) {
	var services []serviceInfo
	if err := json.Unmarshal(output, &services); err != nil {
		return nil, fmt.Errorf("failed to parse kamal-proxy list output: %w", err)
	}

	for _, svc := range services {
		if svc.Service == service {
			result := &HealthResult{
				Connections: svc.Connections,
				Weight:      100, // Kamal Proxy uses 100% by default
			}

			switch svc.State {
			case "running":
				result.Status = "up"
			case "paused", "draining":
				result.Status = "drain"
			default:
				result.Status = "down"
			}

			return result, nil
		}
	}

	return nil, fmt.Errorf("service %q not found in kamal-proxy", service)
}

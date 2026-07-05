package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/AmadlaOrg/waiter-proxy-kamal/kamal"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	appName = "waiter-proxy-kamal"
	version = "1.0.0"
)

var (
	endpoint    string
	serviceName string
)

var rootCmd = &cobra.Command{
	Use:     appName,
	Short:   "Waiter proxy plugin for managing traffic via Kamal Proxy",
	Version: version,
}

var (
	infoOutputFlag string
	infoHeryFlag   bool
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show plugin metadata",
	Run: func(cmd *cobra.Command, args []string) {
		metadata := map[string]any{
			"name":        appName,
			"version":     version,
			"engine":      "kamal-proxy",
			"type":        "proxy",
			"description": "Manages traffic routing via Kamal Proxy",
		}
		if err := writeInfoOutput(os.Stdout, infoOutputFlag, infoHeryFlag, metadata); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding metadata: %v\n", err)
			os.Exit(1)
		}
	},
}

type heryEnvelope struct {
	Type string `json:"_type" yaml:"_type"`
	Body any    `json:"_body" yaml:"_body"`
}

func writeInfoOutput(w io.Writer, format string, hery bool, data map[string]any) error {
	if hery {
		return writeHeryOutput(w, format, data)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		bytes, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(bytes))
		return err
	default:
		table := tablewriter.NewWriter(w)
		table.Header("Field", "Value")
		table.Append("Name", fmt.Sprint(data["name"]))
		table.Append("Version", fmt.Sprint(data["version"]))
		table.Append("Engine", fmt.Sprint(data["engine"]))
		if t, ok := data["type"]; ok {
			table.Append("Type", fmt.Sprint(t))
		}
		table.Append("Description", fmt.Sprint(data["description"]))
		if supports, ok := data["supports"].([]string); ok {
			table.Append("Supports", strings.Join(supports, "\n"))
		}
		table.Render()
		return nil
	}
}

func writeHeryOutput(w io.Writer, format string, data map[string]any) error {
	envelope := heryEnvelope{
		Type: "amadla.org/entity/tools/info@v1.0.0",
		Body: data,
	}

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(envelope)
	case "table":
		fmt.Fprintf(w, "_type: %s\n\n", envelope.Type)
		table := tablewriter.NewWriter(w)
		table.Header("Field", "Value")
		table.Append("Name", fmt.Sprint(data["name"]))
		table.Append("Version", fmt.Sprint(data["version"]))
		table.Append("Engine", fmt.Sprint(data["engine"]))
		if t, ok := data["type"]; ok {
			table.Append("Type", fmt.Sprint(t))
		}
		table.Append("Description", fmt.Sprint(data["description"]))
		if supports, ok := data["supports"].([]string); ok {
			table.Append("Supports", strings.Join(supports, "\n"))
		}
		table.Render()
		return nil
	default:
		bytes, err := yaml.Marshal(envelope)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(bytes))
		return err
	}
}

var (
	registerWeight int
	registerTLS    bool

	registerCmd = &cobra.Command{
		Use:   "register <server> <address>",
		Short: "Register a target with Kamal Proxy",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			server := args[0]
			address := args[1]

			host, portStr, err := net.SplitHostPort(address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid address %q: %v\n", address, err)
				os.Exit(2)
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid port %q: %v\n", portStr, err)
				os.Exit(2)
			}

			mgr := kamal.New(endpoint)
			if err := mgr.Register(serviceName, server, host, port, registerWeight); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Target %q registered for service %q\n", server, serviceName)
			return nil
		},
	}
)

var removeCmd = &cobra.Command{
	Use:   "remove <server>",
	Short: "Remove a target from Kamal Proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := kamal.New(endpoint)
		if err := mgr.Remove(serviceName, args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Target %q removed from service %q\n", args[0], serviceName)
		return nil
	},
}

var (
	shiftWeight int

	shiftCmd = &cobra.Command{
		Use:   "shift <server>",
		Short: "Adjust traffic weight for a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := kamal.New(endpoint)
			if err := mgr.Shift(serviceName, args[0], shiftWeight); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Target %q weight set to %d for service %q\n", args[0], shiftWeight, serviceName)
			return nil
		},
	}
)

var (
	drainTimeout int

	drainCmd = &cobra.Command{
		Use:   "drain <server>",
		Short: "Drain connections from a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := kamal.New(endpoint)
			if err := mgr.Drain(serviceName, args[0], drainTimeout); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Target %q drained for service %q\n", args[0], serviceName)
			return nil
		},
	}
)

var healthCmd = &cobra.Command{
	Use:   "health <server>",
	Short: "Check health of a target",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr := kamal.New(endpoint)
		result, err := mgr.Health(serviceName, args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding result: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", "http://127.0.0.1:80", "Kamal Proxy endpoint")
	rootCmd.PersistentFlags().StringVar(&serviceName, "service", "web", "Service name")

	registerCmd.Flags().IntVar(&registerWeight, "weight", 100, "Target weight")
	registerCmd.Flags().BoolVar(&registerTLS, "tls", false, "Enable TLS for target")
	shiftCmd.Flags().IntVar(&shiftWeight, "weight", 100, "Target weight")
	drainCmd.Flags().IntVar(&drainTimeout, "timeout", 30, "Drain timeout in seconds")

	infoCmd.Flags().StringVarP(&infoOutputFlag, "output", "o", "table", "Output format: table, json, yaml")
	infoCmd.Flags().BoolVar(&infoHeryFlag, "hery", false, "Wrap output in HERY envelope (_type, _body)")

	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(shiftCmd)
	rootCmd.AddCommand(drainCmd)
	rootCmd.AddCommand(healthCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

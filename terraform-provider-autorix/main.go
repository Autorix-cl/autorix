package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

var version = "dev"

func runCLI(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("terraform-provider-autorix", flag.ContinueOnError)
	fs.SetOutput(errOut)
	showVersion := fs.Bool("version", false, "Print provider version")
	showSchema := fs.Bool("schema", false, "Print supported Terraform resource schema JSON")
	checkConn := fs.Bool("check", false, "Verify connectivity to the configured Autorix control-plane endpoint")
	endpoint := fs.String("endpoint", "", "Autorix base endpoint URL (overrides AUTORIX_ENDPOINT)")
	token := fs.String("token", "", "Autorix API or internal auth token (overrides AUTORIX_API_TOKEN)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		_, _ = fmt.Fprintf(out, "terraform-provider-autorix %s\n", version)
		return 0
	}

	if *showSchema {
		p := &Provider{}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(p.Schema()); err != nil {
			_, _ = fmt.Fprintf(errOut, "encode schema: %v\n", err)
			return 1
		}
		return 0
	}

	if *checkConn {
		p, err := NewProvider(Config{Endpoint: *endpoint, APIToken: *token})
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "configuration error: %v\n", err)
			return 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.Ping(ctx); err != nil {
			_, _ = fmt.Fprintf(errOut, "control-plane health check failed: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(out, "ok")
		return 0
	}

	_, _ = fmt.Fprintf(out, "terraform-provider-autorix %s — use -schema, -check, or -version\n", version)
	return 0
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

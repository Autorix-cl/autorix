package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// version is stamped by the release workflow with -ldflags -X main.version.
var version = "dev"

var errApplyNotImplemented = errors.New("declarative apply is not yet implemented against the control-plane API")

type Config struct {
	ArgusURL   string
	NexusURL   string
	Token      string
	HTTPClient *http.Client
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := Config{
		ArgusURL: getEnv("AUTORIX_ARGUS_URL", "http://localhost:4400"),
		NexusURL: getEnv("AUTORIX_NEXUS_URL", "http://localhost:8080"),
		Token:    getEnv("AUTORIX_TOKEN", ""),
	}

	command := os.Args[1]
	args := os.Args[2:]

	var err error
	switch command {
	case "version":
		fmt.Printf("autorixctl version %s (Go 1.25)\n", version)
	case "fleet":
		err = runFleet(cfg, args, os.Stdout)
	case "token":
		err = runToken(cfg, args, os.Stdout)
	case "check":
		err = runCheck(cfg, args, os.Stdout)
	case "audit":
		err = runAudit(cfg, args, os.Stdout)
	case "apply":
		err = runApply(cfg, args, os.Stdout)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`autorixctl - Unified CLI for the Autorix Zero-Trust & IAM Control Plane

Usage:
  autorixctl <command> [arguments]

Commands:
  fleet list                     List all registered engine instances and health
  token mint --engine <type>     Mint a new instance enrollment token
  check --ns <n> --obj <o>       Evaluate a Zanzibar ReBAC permission check
        --rel <r> --subj <s>
  audit list                     Query the immutable audit trail
  audit verify                   Cryptographically verify audit SHA-256 hash chain
  apply -f <file.yaml>           Validate declarative configuration (control-plane apply pending)
  version                        Print autorixctl version

Environment Variables:
  AUTORIX_ARGUS_URL              Argus Control Plane URL (default: http://localhost:4400)
  AUTORIX_NEXUS_URL              Nexus ReBAC Engine URL (default: http://localhost:8080)
  AUTORIX_TOKEN                  Admin operator JWT or API key`)
}

func runFleet(cfg Config, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "list" {
		return errors.New("usage: autorixctl fleet list [--env <env>]")
	}

	fs := flag.NewFlagSet("fleet list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	env := fs.String("env", "", "Filter by environment")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/v1/instances", strings.TrimRight(cfg.ArgusURL, "/"))
	if *env != "" {
		q := url.Values{}
		q.Set("env", *env)
		endpoint += "?" + q.Encode()
	}

	resp, err := doRequest(cfg, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("querying fleet: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading fleet response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fleet request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	_, _ = fmt.Fprintln(out, string(body))
	return nil
}

func runToken(cfg Config, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "mint" {
		return errors.New("usage: autorixctl token mint --engine <type> [--env <env>] [--description <desc>]")
	}

	fs := flag.NewFlagSet("token mint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	engineType := fs.String("engine", "", "Engine type (nexus, ego, janus, vulcan, hermes, themis, aegis)")
	env := fs.String("env", "production", "Environment name")
	desc := fs.String("description", "autorixctl minted token", "Token description")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	if *engineType == "" {
		return errors.New("--engine is required")
	}

	endpoint := fmt.Sprintf("%s/v1/enrollment-tokens", strings.TrimRight(cfg.ArgusURL, "/"))
	payload := map[string]string{
		"engine_type": *engineType,
		"environment": *env,
		"description": *desc,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding token payload: %w", err)
	}

	resp, err := doRequest(cfg, http.MethodPost, endpoint, data)
	if err != nil {
		return fmt.Errorf("minting token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading token response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token mint failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	_, _ = fmt.Fprintln(out, string(body))
	return nil
}

func runCheck(cfg Config, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	ns := fs.String("ns", "", "Namespace (e.g. document)")
	obj := fs.String("obj", "", "Object ID (e.g. doc_1)")
	rel := fs.String("rel", "", "Relation (e.g. viewer, editor)")
	subj := fs.String("subj", "", "Subject (e.g. alice, user:123)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *ns == "" || *obj == "" || *rel == "" || *subj == "" {
		return errors.New("usage: autorixctl check --ns <namespace> --obj <object> --rel <relation> --subj <subject>")
	}

	endpoint := fmt.Sprintf("%s/check", strings.TrimRight(cfg.NexusURL, "/"))
	subjNS := "user"
	subjID := *subj
	if parts := strings.SplitN(*subj, ":", 2); len(parts) == 2 {
		subjNS = parts[0]
		subjID = parts[1]
	}
	payload := map[string]interface{}{
		"namespace":         *ns,
		"object":            *obj,
		"relation":          *rel,
		"subject_namespace": subjNS,
		"subject_id":        subjID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encoding check payload: %w", err)
	}

	resp, err := doRequest(cfg, http.MethodPost, endpoint, data)
	if err != nil {
		return fmt.Errorf("checking permission: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading check response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("check request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res map[string]interface{}
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("invalid JSON response from Nexus: %w", err)
	}

	allowed, ok := res["allowed"].(bool)
	if !ok {
		return errors.New("unexpected response from Nexus: missing boolean 'allowed' field")
	}

	if allowed {
		_, _ = fmt.Fprintln(out, "ALLOW: Subject has relation on object")
	} else {
		_, _ = fmt.Fprintln(out, "DENY: Permission denied")
	}
	return nil
}

func runAudit(cfg Config, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: autorixctl audit [list|verify]")
	}

	sub := args[0]
	var endpoint string
	switch sub {
	case "verify":
		endpoint = fmt.Sprintf("%s/admin/audit/verify", strings.TrimRight(cfg.ArgusURL, "/"))
	case "list":
		endpoint = fmt.Sprintf("%s/admin/audit", strings.TrimRight(cfg.ArgusURL, "/"))
	default:
		return fmt.Errorf("unknown audit subcommand: %s", sub)
	}

	resp, err := doRequest(cfg, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("audit %s failed: %w", sub, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading audit %s response: %w", sub, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("audit %s failed (%d): %s", sub, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if !json.Valid(body) {
		return fmt.Errorf("audit %s returned invalid JSON response", sub)
	}

	_, _ = fmt.Fprintln(out, string(body))
	return nil
}

func runApply(_ Config, args []string, _ io.Writer) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	file := fs.String("f", "", "File path to apply (.yaml or .json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *file == "" {
		return errors.New("usage: autorixctl apply -f <configuration.yaml>")
	}

	content, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("reading file %q: %w", *file, err)
	}

	if err := validateConfigFile(*file, content); err != nil {
		return fmt.Errorf("invalid configuration file %q: %w", *file, err)
	}

	return fmt.Errorf("%w (validated %s; no changes were applied)", errApplyNotImplemented, *file)
}

func validateConfigFile(path string, content []byte) error {
	if !utf8.Valid(content) {
		return errors.New("file must be valid UTF-8")
	}
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return errors.New("configuration file is empty")
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		var parsed interface{}
		if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
			return fmt.Errorf("invalid JSON syntax: %w", err)
		}
		return nil

	case ".yaml", ".yml":
		// If valid JSON (JSON is a subset of YAML 1.2), accept it directly.
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			var parsed interface{}
			if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
				return fmt.Errorf("invalid JSON/YAML syntax: %w", err)
			}
			return nil
		}
		hasMapping := false
		for lineNum, rawLine := range strings.Split(trimmed, "\n") {
			line := strings.TrimRight(rawLine, "\r")
			stripped := strings.TrimSpace(line)
			if stripped == "" || strings.HasPrefix(stripped, "#") || stripped == "---" || stripped == "..." {
				continue
			}
			// Leading tab indentation is forbidden by the YAML specification.
			leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			if strings.ContainsRune(leading, '\t') {
				return fmt.Errorf("line %d: tab indentation is not valid YAML", lineNum+1)
			}
			if idx := strings.IndexByte(stripped, ':'); idx > 0 {
				hasMapping = true
			}
		}
		if !hasMapping {
			return errors.New("YAML file contains no valid key-value definitions")
		}
		return nil

	default:
		return fmt.Errorf("unsupported file extension %q (expected .yaml, .yml, or .json)", ext)
	}
}

func doRequest(cfg Config, method, endpoint string, body []byte) (*http.Response, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, endpoint, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}

	return client.Do(req)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

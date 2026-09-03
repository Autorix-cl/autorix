package main

import (
	"flag"
	"fmt"
	"os"

	autortls "github.com/autorix/platform/tls"
)

func main() {
	outDir := flag.String("out", "certs", "Output directory for generated certificates")
	flag.Parse()

	services := []string{
		"nexus",
		"vulcan",
		"ego",
		"janus",
		"aegis",
		"themis",
		"hermes",
		"argus",
		"console",
	}

	fmt.Printf("Generating Autorix Service Mesh PKI in %q...\n", *outDir)
	if err := autortls.GenerateMeshPKI(*outDir, services); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating mesh PKI: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully generated:")
	fmt.Printf("  - %s/ca.crt and ca.key\n", *outDir)
	for _, svc := range services {
		fmt.Printf("  - %s/%s.crt and %s.key (DNS:%s, SPIFFE:spiffe://autorix.internal/service/%s)\n", *outDir, svc, svc, svc, svc)
	}
}

// Package main provides the VIGIL command-line interface
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("VIGIL - Virtual Integrated Military Infrastructure for LVC")
	fmt.Println("Version: v0.0.1")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  vigil start       Start all services")
	fmt.Println("  vigil stop        Stop all services")
	fmt.Println("  vigil status      Show service status")
	fmt.Println("  vigil test        Run integration tests")
	fmt.Println("")
	fmt.Println("Services:")
	fmt.Println("  opir-ingest       OPIR data ingestion")
	fmt.Println("  sensor-fusion     Multi-source track fusion")
	fmt.Println("  missile-warning   Threat warning engine")
	fmt.Println("  alert-dissem      Alert dissemination")
	fmt.Println("  lvc-coord         LVC coordinator")
	fmt.Println("  replay            Mission replay")
	fmt.Println("")
	os.Exit(0)
}
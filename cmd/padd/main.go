package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	appName      = "PADD"
	appVersion   = "0.1.0"
	resourcesDir = "resources"
)

// getDataDirectory determines the data directory using a tiered approach:
// 1. Command-line flag (-data) takes highest precedence.
// 2. Environment variable PADD_DATA_DIR if flag is not set.
// 3. XDG_DATA_HOME/padd or $HOME/.local/share/padd as fallback.
func getDataDirectory(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if envDir := os.Getenv("PADD_DATA_DIR"); envDir != "" {
		return envDir, nil
	}

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user home directory: %v", err)
		}
		xdgDataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(xdgDataHome, "padd"), nil
}

func main() {
	var port int
	var addr string
	var dataFlag string
	var showVersion bool

	// Note to self about Flag aliases: Go's flag package allows multiple flag names to point to the same variable.
	// When you call BoolVar/StringVar/etc. multiple times with the same variable pointer,
	// you create aliases that all modify the same memory location. This enables both short
	// and long flag versions (e.g., -v and -version) without needing separate variables.
	// The default value is only applied once, not overridden - both flags share the same
	// default and will set the same variable when used by the user.
	flagSet := flag.NewFlagSet(appName, flag.ExitOnError)
	flagSet.StringVar(&dataFlag, "data", "", "Directory to store markdown files.")
	flagSet.StringVar(&dataFlag, "d", "", "Directory to store markdown files.")
	flagSet.IntVar(&port, "port", 8080, "Port to run the server on.")
	flagSet.IntVar(&port, "p", 8080, "Port to run the server on.")
	flagSet.StringVar(&addr, "addr", "localhost", "Address to bind the server to.")
	flagSet.StringVar(&addr, "a", "localhost", "Address to bind the server to.")
	flagSet.BoolVar(&showVersion, "version", false, "Show application version.")
	flagSet.BoolVar(&showVersion, "v", false, "Show application version.")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintf(flagSet.Output(), "PADD - Personal Assistant for Daily Documentation\n\n")
		flagSet.PrintDefaults()
	}

	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(fmt.Errorf("error parsing flags: %v", err))
	}

	if showVersion {
		fmt.Printf("PADD version %s\n", appVersion)
		os.Exit(0)
		return
	}

	resolvedDataDir, err := getDataDirectory(dataFlag)
	if err != nil {
		log.Fatal(fmt.Errorf("error determining data directory: %v", err))
	}

	// Create a context for the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := NewServer(ctx, resolvedDataDir)
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing server: %v", err))
	}

	// Start the server (blocking call)
	if err := server.Start(addr, port); err != nil {
		log.Fatal(err)
	}
}

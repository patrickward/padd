package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/patrickward/padd"
	"github.com/patrickward/padd/internal/version"
)

const (
	appName           = "PADD"
	resourcesDir      = "resources"
	envPaddData       = "PADD_DATA_DIR"
	envPaddKeys       = "PADD_KEYS_DIR"
	envPaddIdentities = "PADD_IDENTITIES_FILE"
	envPaddRecipients = "PADD_RECIPIENTS_FILE"
)

// getXDGDataHome determines the XDG_DATA_HOME directory.
func getXDGDataHome() (string, error) {
	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user home directory: %v", err)
		}
		xdgDataHome = filepath.Join(homeDir, ".local", "share")
	}

	return xdgDataHome, nil
}

// getConfigDataDirectory determines the data directory using a tiered approach:
// 1. The command-line flag value (e.g., -some-flag) takes the highest precedence.
// 2. The environment variable (e.g., PADD_SOME_VAR) if a flag is not set.
// 3. XDG_DATA_HOME/padd/subdirectory as a fallback.
func getConfigDataDirectory(flagValue, envVar, subdirectory string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue, nil
	}

	xdgDataHome, err := getXDGDataHome()
	if err != nil {
		return "", fmt.Errorf("unable to determine XDG_DATA_HOME: %v", err)
	}

	return filepath.Join(xdgDataHome, "padd", subdirectory), nil
}

// getConfigValue determines the value using a tiered approach:
// 1. The command-line flag value (e.g., -some-flag) takes the highest precedence.
// 2. The environment variable PADD_<ENV_VAR> if a flag is not set.
// 3. The default value as a fallback.
func getConfigValue(flagValue, envVar, defaultValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envValue := os.Getenv(envVar); envValue != "" {
		return envValue
	}

	return defaultValue
}

// getDefaultKeys returns any default keys found in the data directory
// If there is a keys directory, and it contains a key.pub and key.txt file,
// those files will be returned as the default keys. Otherwise, an empty list is returned.
// (e.g., ~/.local/share/padd/keys/key.pub and ~/.local/share/padd/keys/key.txt).
func getDefaultKeys(keysDir string) (identitiesFile, recipientsFile string) {
	// Check if the data directory exists
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		log.Printf("the keys directory %s does not exist", keysDir)
		return "", ""
	}

	// Check if the key files exist
	identitiesFile = filepath.Join(keysDir, "key.txt")
	recipientsFile = filepath.Join(keysDir, "key.pub")

	if _, err := os.Stat(identitiesFile); os.IsNotExist(err) {
		log.Printf("default identities key file %s does not exist", identitiesFile)
		return "", ""
	}

	if _, err := os.Stat(recipientsFile); os.IsNotExist(err) {
		log.Printf("default recipients key file %s does not exist", recipientsFile)
		return "", ""
	}

	return identitiesFile, recipientsFile
}

func main() {
	var port int
	var addr string
	var dataDirFlag string
	var keysDirFlag string
	var identitiesFile string
	var recipientsFile string
	var generateKeys bool
	var showVersion bool

	// Note to self about Flag aliases: Go's flag package allows multiple flag names to point to the same variable.
	// When you call BoolVar/StringVar/etc. multiple times with the same variable pointer,
	// you create aliases that all modify the same memory location. This enables both short
	// and long flag versions (e.g., -v and -version) without needing separate variables.
	// The default value is only applied once, not overridden - both flags share the same
	// default and will set the same variable when used by the user.
	flagSet := flag.NewFlagSet(appName, flag.ExitOnError)
	flagSet.StringVar(&dataDirFlag, "data", "", "Directory to store markdown files.")
	flagSet.StringVar(&dataDirFlag, "d", "", "Directory to store markdown files.")
	flagSet.StringVar(&keysDirFlag, "keys-dir", "", "Directory for key operations (generation, etc.)")
	flagSet.StringVar(&keysDirFlag, "k", "", "Directory for key operations (generation, etc.)")
	flagSet.StringVar(&identitiesFile, "identity", "", "Use the identity file at the specified path for decryption.")
	flagSet.StringVar(&identitiesFile, "i", "", "Use the identity file at the specified path for decryption.")
	flagSet.StringVar(&recipientsFile, "recipient", "", "Use the recipient file at the specified path for encryption.")
	flagSet.StringVar(&recipientsFile, "r", "", "Use the recipient file at the specified path for encryption.")
	flagSet.BoolVar(&generateKeys, "generate-keys", false, "Generate a new key pair and save to keys-dir.")
	flagSet.BoolVar(&generateKeys, "g", false, "Generate a new key pair and save to keys-dir.")

	flagSet.IntVar(&port, "port", 8080, "Port to run the server on.")
	flagSet.IntVar(&port, "p", 8080, "Port to run the server on.")
	flagSet.StringVar(&addr, "addr", "localhost", "Address to bind the server to.")
	flagSet.StringVar(&addr, "a", "localhost", "Address to bind the server to.")

	flagSet.BoolVar(&showVersion, "version", false, "Show application version.")
	flagSet.BoolVar(&showVersion, "v", false, "Show application version.")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintf(flagSet.Output(), "PADD - Personal Assistant for Daily Documentation\n\n")
		_, _ = fmt.Fprintf(flagSet.Output(), "Examples:\n")
		_, _ = fmt.Fprintf(flagSet.Output(), "  # Generate new keys:\n")
		_, _ = fmt.Fprintf(flagSet.Output(), "  %s -generate-keys -keys-dir ~/.padd/keys\n\n", appName)
		_, _ = fmt.Fprintf(flagSet.Output(), "  # Use specific identity and recipient:\n")
		_, _ = fmt.Fprintf(flagSet.Output(), "  %s -identity ~/.padd/keys/key.txt -recipient ~/.padd/keys/key.pub...\n\n", appName)
		_, _ = fmt.Fprintf(flagSet.Output(), "  # Use YubiKey plugin:\n")
		_, _ = fmt.Fprintf(flagSet.Output(), "  %s -identity ~/.age/yubikey-identities.txt -recipient ~/.padd/keys/key.pub...\n\n", appName)
		_, _ = fmt.Fprintf(flagSet.Output(), "Options:\n")
		flagSet.PrintDefaults()
	}

	// Parse the flags
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(fmt.Errorf("error parsing flags: %v", err))
	}

	if showVersion {
		fmt.Printf("version: %s\n", version.Get())
		os.Exit(0)
		return
	}

	// Resolve the keys directory.
	keysDir, err := getConfigDataDirectory(keysDirFlag, envPaddKeys, "keys")
	if err != nil {
		log.Fatal(fmt.Errorf("error determining keys directory: %v", err))
	}

	// Generate new keys - outputs to timestamped key pair files in the keys directory.
	if generateKeys {
		if err != nil {
			log.Fatal(fmt.Errorf("error determining keys directory: %v", err))
		}

		publicKey, _, publicPath, privatePath, err := padd.GenerateNewEncryptionPair(keysDir)
		if err != nil {
			log.Fatal(fmt.Errorf("error generating new encryption identity: %v", err))
		}

		fmt.Printf("Generated new encryption identity:\n")
		fmt.Printf("  Public key: %s\n", publicKey)
		fmt.Printf("  Public key file: %s\n", publicPath)
		fmt.Printf("  Private key file: %s\n", privatePath)
		fmt.Printf("\nTo use these keys:\n")
		fmt.Printf("  %s -identity %s -recipient %s\n", appName, privatePath, publicKey)
		os.Exit(0)
		return
	}

	// Resolve the data directory.
	dataDir, err := getConfigDataDirectory(dataDirFlag, envPaddData, "data")
	if err != nil {
		log.Fatal(fmt.Errorf("error determining data directory: %v", err))
	}

	// Set up the encryption config
	encryptionManager := padd.NewEncryptionManager()
	identitiesFile = getConfigValue(identitiesFile, envPaddIdentities, "")
	recipientsFile = getConfigValue(recipientsFile, envPaddRecipients, "")
	if identitiesFile == "" || recipientsFile == "" {
		identitiesFile, recipientsFile = getDefaultKeys(keysDir)
	}

	if err = encryptionManager.LoadEncryptionKeys(identitiesFile, recipientsFile); err != nil {
		log.Printf("Error loading encryption keys: %v", err)
		log.Printf("Encryption disabled!")
	} else {
		log.Printf("Encryption enabled!")
	}

	// Create a context for the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the server and start it
	server, err := NewServer(ctx, dataDir, WithEncryptionManager(encryptionManager))
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing server: %v", err))
	}

	err = server.Start(addr, port)
	if err != nil {
		log.Fatal(err)
	}
}

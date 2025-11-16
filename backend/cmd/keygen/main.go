package main

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/quantarax/backend/internal/crypto"
	"golang.org/x/term"
)

const (
	identityKeyFile = "identity.key"
	identityPubFile = "identity.pub"
)

var (
	// Global flags
	outputDir     string
	noPassphrase  bool
	force         bool
	includePrivate bool
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "generate":
		generateCmd(args)
	case "show":
		showCmd(args)
	case "export":
		exportCmd(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("keygen - QuantaraX Key Management Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  keygen generate [flags]  - Generate new identity keypair")
	fmt.Println("  keygen show              - Display public key information")
	fmt.Println("  keygen export [flags]    - Export keys for backup")
	fmt.Println()
	fmt.Println("Run 'keygen <command> -h' for command-specific help")
}

func generateCmd(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	fs.StringVar(&outputDir, "output-dir", crypto.GetDefaultKeystorePath(), "Key storage directory")
	fs.BoolVar(&noPassphrase, "no-passphrase", false, "Generate without passphrase protection")
	fs.BoolVar(&force, "force", false, "Overwrite existing keys")
	fs.Parse(args)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Check if keys already exist
	keyPath := filepath.Join(outputDir, identityKeyFile)
	pubPath := filepath.Join(outputDir, identityPubFile)

	if !force {
		if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
			fmt.Println("Identity keys already exist.")
			fmt.Print("Overwrite existing keys? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Aborted.")
				return
			}
		}
	}

	fmt.Println("Generating new identity keypair...")
	fmt.Println()

	// Generate Ed25519 keypair
	kp, err := crypto.GenerateEd25519()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate keypair: %v\n", err)
		os.Exit(1)
	}

	// Get passphrase
	var passphrase string
	if !noPassphrase {
		fmt.Print("Enter passphrase (leave empty for no encryption): ")
		passphraseBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read passphrase: %v\n", err)
			os.Exit(1)
		}
		passphrase = string(passphraseBytes)

		if passphrase != "" {
			fmt.Print("Confirm passphrase: ")
			confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read passphrase: %v\n", err)
				os.Exit(1)
			}

			if passphrase != string(confirmBytes) {
				fmt.Fprintln(os.Stderr, "Passphrases do not match.")
				os.Exit(1)
			}
		}
	}

	// Save private key
	err = crypto.SaveKey(kp.PrivateKey, keyPath, passphrase)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save private key: %v\n", err)
		os.Exit(1)
	}

	// Save public key
	pubKeyB64 := base64.StdEncoding.EncodeToString(kp.PublicKey)
	err = os.WriteFile(pubPath, []byte(pubKeyB64+"\n"), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save public key: %v\n", err)
		os.Exit(1)
	}

	// Compute fingerprint
	hash := sha256.Sum256(kp.PublicKey)
	fingerprint := fmt.Sprintf("SHA256:%x", hash[:8])

	fmt.Println("Identity keypair generated successfully!")
	fmt.Println()
	fmt.Println("Public Key:")
	fmt.Printf("  %s\n", pubKeyB64)
	fmt.Println()
	fmt.Println("Fingerprint:")
	fmt.Printf("  %s\n", fingerprint)
	fmt.Println()
	fmt.Println("Keys stored in:")
	fmt.Printf("  %s\n", outputDir)

	if passphrase == "" {
		fmt.Println()
		fmt.Println("WARNING: Keys stored WITHOUT encryption (insecure)")
	}
}

func showCmd(args []string) {
	fs := flag.NewFlagSet("show", flag.ExitOnError)
	fs.StringVar(&outputDir, "keys-dir", crypto.GetDefaultKeystorePath(), "Key storage directory")
	fs.Parse(args)

	pubPath := filepath.Join(outputDir, identityPubFile)

	// Read public key
	pubKeyData, err := os.ReadFile(pubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read public key: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'keygen generate' first to create keys")
		os.Exit(1)
	}

	pubKeyB64 := string(pubKeyData)
	pubKeyB64 = pubKeyB64[:len(pubKeyB64)-1] // Remove trailing newline

	// Decode for fingerprint
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode public key: %v\n", err)
		os.Exit(1)
	}

	// Compute fingerprint
	hash := sha256.Sum256(pubKeyBytes)
	fingerprint := fmt.Sprintf("SHA256:%x", hash[:8])

	// Get file info for creation time
	fileInfo, _ := os.Stat(pubPath)
	modTime := fileInfo.ModTime().Format(time.RFC3339)

	fmt.Println("Identity Public Key:")
	fmt.Printf("  %s\n", pubKeyB64)
	fmt.Println()
	fmt.Println("Fingerprint:")
	fmt.Printf("  %s\n", fingerprint)
	fmt.Println()
	fmt.Println("Key Type: Ed25519")
	fmt.Printf("Created: %s\n", modTime)
}

func exportCmd(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	fs.StringVar(&outputDir, "keys-dir", crypto.GetDefaultKeystorePath(), "Key storage directory")
	fs.BoolVar(&includePrivate, "include-private", false, "Include private key in export")
	fs.Parse(args)

	pubPath := filepath.Join(outputDir, identityPubFile)

	// Read public key
	pubKeyData, err := os.ReadFile(pubPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read public key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Public Key:")
	fmt.Print(string(pubKeyData))

	if includePrivate {
		fmt.Println()
		fmt.Println("WARNING: Exporting private key is sensitive operation")
		fmt.Println("Private key export not yet implemented in this version")
		fmt.Println("Use the keystore file directly for backup")
	}
}
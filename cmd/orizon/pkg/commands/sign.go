// Package commands provides the sign command implementation for package management.
// This handles package signature creation and verification.
package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// SignCommand handles package signing operations.
// It creates cryptographic signatures for packages to ensure integrity and authenticity.
type SignCommand struct {
	*BaseCommand
}

// NewSignCommand creates a new sign command handler.
func NewSignCommand() *SignCommand {
	return &SignCommand{
		BaseCommand: NewBaseCommand(
			"Sign a package with cryptographic signature",
			"usage: orizon pkg sign --cid <cid> [--subject <subject>]",
		),
	}
}

// Execute implements the CommandHandler interface for sign operations.
func (c *SignCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	cidStr := fs.String("cid", "", "content ID to sign")
	subject := fs.String("subject", "dev", "certificate subject")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse sign flags: %w", err)
	}

	if *cidStr == "" {
		return fmt.Errorf("usage: orizon pkg sign --cid <cid> [--subject <subject>]")
	}

	// Generate ephemeral keypair for signing
	pub, priv, err := packagemanager.GenerateEd25519Keypair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Create self-signed root certificate
	root, err := packagemanager.SelfSignRoot(*subject, pub, priv, 24*60*60*365*10)
	if err != nil {
		return fmt.Errorf("failed to create root certificate: %w", err)
	}

	// Sign the package
	bundle, err := packagemanager.SignPackage(context.Background(), ctx.Registry, packagemanager.CID(*cidStr), priv, []packagemanager.Certificate{root}, ctx.SignatureStore)
	if err != nil {
		return fmt.Errorf("failed to sign package: %w", err)
	}

	fmt.Printf("signed %s with key %s (chain len %d)\n", *cidStr, bundle.KeyID, len(bundle.Chain))
	return nil
}

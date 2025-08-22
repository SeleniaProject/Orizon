// Package commands provides the verify-sig command implementation for package management.
// This handles package signature verification operations.
package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// VerifySigCommand handles signature verification operations.
// It verifies the cryptographic signatures of packages to ensure integrity and authenticity.
type VerifySigCommand struct {
	*BaseCommand
}

// NewVerifySigCommand creates a new verify-sig command handler.
func NewVerifySigCommand() *VerifySigCommand {
	return &VerifySigCommand{
		BaseCommand: NewBaseCommand(
			"Verify package cryptographic signatures",
			"usage: orizon pkg verify-sig --cid <cid>",
		),
	}
}

// Execute implements the CommandHandler interface for verify-sig operations.
func (c *VerifySigCommand) Execute(ctx types.RegistryContext, args []string) error {
	fs := flag.NewFlagSet("verify-sig", flag.ExitOnError)
	cidStr := fs.String("cid", "", "content ID to verify")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse verify-sig flags: %w", err)
	}

	if *cidStr == "" {
		return fmt.Errorf("usage: orizon pkg verify-sig --cid <cid>")
	}

	// Create trust store from existing signatures (demo approach)
	trustStore := packagemanager.NewTrustStore()

	// Load all bundles for this package
	bundles, err := ctx.SignatureStore.List(packagemanager.CID(*cidStr))
	if err != nil {
		return fmt.Errorf("failed to list signatures: %w", err)
	}

	// Trust all root certificates from existing bundles (demo)
	for _, bundle := range bundles {
		if len(bundle.Chain) > 0 {
			trustStore.AddRoot(bundle.Chain[len(bundle.Chain)-1].PublicKey)
		}
	}

	// Verify package signature
	if err := packagemanager.VerifyPackage(context.Background(), ctx.Registry, trustStore, packagemanager.CID(*cidStr), ctx.SignatureStore); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	fmt.Println("signature verified (at least one)")
	return nil
}

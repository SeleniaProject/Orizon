// Package utils provides registry-related utility functions for package management.
// These functions handle registry initialization, authentication, and connection management.
package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orizon-lang/orizon/cmd/orizon/pkg/types"
	"github.com/orizon-lang/orizon/internal/packagemanager"
)

// CreateRegistry creates and initializes a package registry based on environment configuration.
// It supports both HTTP and local file-based registries with automatic detection.
func CreateRegistry() (packagemanager.Registry, error) {
	regEnv := strings.TrimSpace(os.Getenv("ORIZON_REGISTRY"))

	var reg packagemanager.Registry
	var err error

	// Check if it's an HTTP registry
	if strings.HasPrefix(strings.ToLower(regEnv), "http://") ||
		strings.HasPrefix(strings.ToLower(regEnv), "https://") {
		// HTTP client will pick ORIZON_REGISTRY_TOKEN automatically or from credentials.json
		reg = packagemanager.NewHTTPRegistry(regEnv)
	} else {
		// Local file registry
		regPath := regEnv
		if regPath == "" {
			regPath = DefaultRegistryPath
		}

		// Ensure registry directory exists
		if err := os.MkdirAll(regPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create registry directory: %w", err)
		}

		reg, err = packagemanager.NewFileRegistry(regPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open registry: %w", err)
		}
	}

	return reg, nil
}

// CreateRegistryContext creates a complete registry context with registry and signature store.
// This is the main entry point for initializing package management operations.
func CreateRegistryContext() (types.RegistryContext, error) {
	registry, err := CreateRegistry()
	if err != nil {
		return types.RegistryContext{}, fmt.Errorf("failed to create registry: %w", err)
	}

	sigStore, err := GetSignatureStore()
	if err != nil {
		return types.RegistryContext{}, fmt.Errorf("failed to create signature store: %w", err)
	}

	return types.RegistryContext{
		Registry:       registry,
		SignatureStore: sigStore,
	}, nil
}

// LoadCredentials loads registry credentials from the default credentials file.
// This supports authentication for package publishing and fetching operations.
func LoadCredentials() (map[string]struct {
	Token string `json:"token"`
}, error) {
	credentialsPath := filepath.Join(".orizon", "credentials.json")

	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty credentials if file doesn't exist
			return make(map[string]struct {
				Token string `json:"token"`
			}), nil
		}
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	var creds struct {
		Registries map[string]struct {
			Token string `json:"token"`
		} `json:"registries"`
	}

	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	if creds.Registries == nil {
		creds.Registries = make(map[string]struct {
			Token string `json:"token"`
		})
	}

	return creds.Registries, nil
}

// SaveCredentials saves registry credentials to the default credentials file.
// This is used after authentication operations to persist tokens.
func SaveCredentials(credentials map[string]struct {
	Token string `json:"token"`
}) error {
	credentialsPath := filepath.Join(".orizon", "credentials.json")

	// Ensure .orizon directory exists
	if err := os.MkdirAll(".orizon", 0755); err != nil {
		return fmt.Errorf("failed to create .orizon directory: %w", err)
	}

	credsData := struct {
		Registries map[string]struct {
			Token string `json:"token"`
		} `json:"registries"`
	}{
		Registries: credentials,
	}

	data, err := json.MarshalIndent(credsData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(credentialsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials: %w", err)
	}

	return nil
}

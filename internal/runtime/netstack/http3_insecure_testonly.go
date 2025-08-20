//go:build test.

package netstack

import "crypto/tls"

// WithInsecureMinTLS12 returns a tls.Config with InsecureSkipVerify for tests.
func WithInsecureMinTLS12() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
}

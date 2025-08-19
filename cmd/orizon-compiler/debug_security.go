package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// デバッグ用のテストプログラム
func debugSecurity() {
	validator := NewSecurityValidator()

	testCases := []string{
		"../../../etc/passwd.oriz",
		"/etc/shadow.oriz",
		"test.oriz",
		strings.Repeat("a/", 1000) + "test.oriz",
		"C:\\Windows\\System32\\cmd.exe",
	}

	for _, testCase := range testCases {
		err := validator.ValidateInputFile(testCase)
		fmt.Printf("Testing: %s\n", testCase)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  OK\n")
		}

		// Clean pathをデバッグ
		cleanPath := filepath.Clean(testCase)
		fmt.Printf("  Clean path: %s\n", cleanPath)

		// パターンマッチング詳細
		for _, pattern := range validator.blockedPatterns {
			if strings.Contains(strings.ToLower(cleanPath), strings.ToLower(pattern)) {
				fmt.Printf("  Matched pattern: %s\n", pattern)
			}
		}
		fmt.Println()
	}
}

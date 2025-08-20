package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type testAttempt struct {
	Outcome string `json:"outcome"`
	Output  string `json:"output"`
	TimeMs  int64  `json:"time_ms"`
}

type testEntry struct {
	Package  string        `json:"package"`
	Name     string        `json:"name"`
	Attempts []testAttempt `json:"attempts"`
}

type junitSummary struct {
	Tests      []testEntry `json:"tests"`
	Total      int         `json:"total"`
	Passed     int         `json:"passed"`
	Failed     int         `json:"failed"`
	Skipped    int         `json:"skipped"`
	DurationMs int64       `json:"duration_ms"`
}

type fuzzStats struct {
	Executions uint64 `json:"executions"`
	Crashes    uint64 `json:"crashes"`
	DurationMs int64  `json:"duration_ms"`
}

func readFileIfPresent(path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, nil
	}

	return b, nil
}

func readCoverageSummary(path string) (string, error) {
	b, err := readFileIfPresent(path)
	if err != nil || len(b) == 0 {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) == 0 {
		return "", nil
	}

	return lines[len(lines)-1], nil
}

func main() {
	var (
		junitPath string
		statsList string
		outPath   string
		title     string
		coverPath string
	)

	flag.StringVar(&junitPath, "junit-summary", "", "path to junit_summary.json")
	flag.StringVar(&statsList, "stats", "", "comma-separated paths to fuzz stats JSON files")
	flag.StringVar(&outPath, "out", "", "optional output markdown path")
	flag.StringVar(&title, "title", "Orizon Smoke Summary", "summary title")
	flag.StringVar(&coverPath, "cover", "", "optional path to coverage summary (cover.txt)")
	flag.Parse()

	var sb strings.Builder

	sb.WriteString("### ")
	sb.WriteString(title)
	sb.WriteString("\n\n")

	if b, err := readFileIfPresent(junitPath); err == nil && len(b) > 0 {
		var js junitSummary
		if err := json.Unmarshal(b, &js); err == nil {
			sb.WriteString("#### Tests\n")
			fmt.Fprintf(&sb, "- total: %d\n- passed: %d\n- failed: %d\n- skipped: %d\n- duration_ms: %d\n\n", js.Total, js.Passed, js.Failed, js.Skipped, js.DurationMs)

			if js.Failed == 0 {
				// show flaky recovered briefly.
				flaky := 0

				for _, t := range js.Tests {
					sawFail := false
					final := ""

					for _, a := range t.Attempts {
						if a.Outcome == "fail" {
							sawFail = true
						}

						final = a.Outcome
					}

					if sawFail && final == "pass" {
						flaky++
					}
				}

				if flaky > 0 {
					fmt.Fprintf(&sb, "- flaky_recovered: %d\n\n", flaky)
				}
			}
		}
	}

	if strings.TrimSpace(statsList) != "" {
		paths := strings.Split(statsList, ",")
		wroteHeader := false

		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}

			b, err := readFileIfPresent(p)
			if err != nil || len(b) == 0 {
				continue
			}

			var fs fuzzStats
			if err := json.Unmarshal(b, &fs); err != nil {
				continue
			}

			if !wroteHeader {
				sb.WriteString("#### Fuzz\n")

				wroteHeader = true
			}

			name := filepath.Base(p)
			// try to derive target from filename e.g., stats_parser.json
			target := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSuffix(name, filepath.Ext(name)), "stats_"), ".json")
			if target == "" {
				target = name
			}

			fmt.Fprintf(&sb, "- %s: executions=%d crashes=%d duration_ms=%d\n", target, fs.Executions, fs.Crashes, fs.DurationMs)
		}

		if wroteHeader {
			sb.WriteString("\n")
		}
	}

	if strings.TrimSpace(coverPath) != "" {
		if last, _ := readCoverageSummary(coverPath); strings.TrimSpace(last) != "" {
			sb.WriteString("#### Coverage\n")
			sb.WriteString("- ")
			sb.WriteString(last)
			sb.WriteString("\n\n")
		}
	}

	out := sb.String()

	if outPath != "" {
		_ = os.MkdirAll(filepath.Dir(outPath), 0o755)
		_ = os.WriteFile(outPath, []byte(out), 0o644)
	}

	fmt.Print(out)
}

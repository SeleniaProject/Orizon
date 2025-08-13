package build

import "testing"

func TestPlatform_Validate(t *testing.T) {
    cases := []Platform{
        {GOOS: "linux", GOARCH: "amd64"},
        {GOOS: "linux", GOARCH: "arm64"},
        {GOOS: "darwin", GOARCH: "amd64"},
        {GOOS: "darwin", GOARCH: "arm64"},
        {GOOS: "windows", GOARCH: "amd64"},
        {GOOS: "windows", GOARCH: "arm64"},
    }
    for _, c := range cases {
        if err := c.Validate(); err != nil { t.Fatalf("unexpected error for %s: %v", c, err) }
    }
    if err := (Platform{}).Validate(); err == nil { t.Fatalf("expected error for empty platform") }
}

func TestGoToolchain_BuildSpec(t *testing.T) {
    tc := GoToolchain{ DefaultFlags: []string{"-trimpath"}, DefaultLdFlags: []string{"-s", "-w"} }
    spec, err := tc.BuildPackage(Platform{GOOS: "linux", GOARCH: "amd64"}, "./cmd/orizon-compiler", "orizon", []string{"-v"}, nil)
    if err != nil { t.Fatalf("build spec error: %v", err) }
    if spec.Cmd != "go" { t.Fatalf("bad cmd: %s", spec.Cmd) }
    if spec.Env["GOOS"] != "linux" || spec.Env["GOARCH"] != "amd64" { t.Fatalf("bad env: %+v", spec.Env) }
    if len(spec.Args) == 0 { t.Fatalf("args empty") }
}



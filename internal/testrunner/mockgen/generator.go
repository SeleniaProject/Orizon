package mockgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// GenOptions controls mock code generation.
type GenOptions struct {
	// Interface to mock
	InterfaceName string
	// Package name of generated code. If empty, use the target package name + "mock" suffix.
	PackageName string
	// Destination path for writing the generated file. If empty, only return the string.
	Destination string
	// Source patterns passed to go/packages (e.g., []string{"./..."} or concrete dirs/files)
	SourcePatterns []string
	// Build tags (comma-separated string or slice). Empty means none.
	BuildTags []string
}

// Generate produces mock code for the specified interface.
func Generate(opts GenOptions) (string, error) {
	if strings.TrimSpace(opts.InterfaceName) == "" {
		return "", errors.New("InterfaceName is required")
	}
	patterns := opts.SourcePatterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax}
	if len(opts.BuildTags) > 0 {
		cfg.BuildFlags = append(cfg.BuildFlags, fmt.Sprintf("-tags=%s", strings.Join(opts.BuildTags, ",")))
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return "", err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return "", errors.New("failed to load packages")
	}

	var (
		foundPkg   *packages.Package
		ifaceType  *types.Interface
		ifaceObj   types.Object
		targetName = opts.InterfaceName
	)
	for _, p := range pkgs {
		// Search for the interface by name in types scope
		if p.Types == nil || p.Types.Scope() == nil {
			continue
		}
		if obj := p.Types.Scope().Lookup(targetName); obj != nil {
			if t, ok := obj.Type().Underlying().(*types.Interface); ok {
				ifaceType = t.Complete()
				ifaceObj = obj
				foundPkg = p
				break
			}
		}
	}
	if foundPkg == nil || ifaceType == nil {
		return "", fmt.Errorf("interface %q not found in provided source patterns", targetName)
	}

	genPkgName := opts.PackageName
	if genPkgName == "" {
		genPkgName = foundPkg.Name + "mock"
	}

	code, err := renderMock(genPkgName, ifaceObj, ifaceType)
	if err != nil {
		return "", err
	}
	if opts.Destination != "" {
		if err := os.MkdirAll(filepath.Dir(opts.Destination), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(opts.Destination, []byte(code), 0o644); err != nil {
			return "", err
		}
	}
	return code, nil
}

func renderMock(pkg string, obj types.Object, iface *types.Interface) (string, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "package %s\n\n", pkg)
	buf.WriteString("import (\n\t\"sync\"\n)\n\n")

	name := obj.Name()
	mockName := name + "Mock"

	// Collect methods including embedded
	methods := collectMethods(iface)

	// Type header
	fmt.Fprintf(&buf, "// %s is a concurrency-safe test double for %s.\n", mockName, name)
	fmt.Fprintf(&buf, "type %s struct {\n\tmu sync.Mutex\n", mockName)
	for _, m := range methods {
		fmt.Fprintf(&buf, "\t%sStub func(%s) (%s)\n", m.name, joinFieldList(m.params), joinFieldList(m.results))
		fmt.Fprintf(&buf, "\t%sCalls []%s_%sCall\n", m.name, name, m.name)
	}
	buf.WriteString("}\n\n")

	// Call records
	for _, m := range methods {
		fmt.Fprintf(&buf, "type %s_%sCall struct { %s %s }\n\n", name, m.name, fieldsDecl("Arg", m.params), fieldsDecl("Ret", m.results))
	}

	// Methods implementations
	for _, m := range methods {
		fmt.Fprintf(&buf, "func (m *%s) %s(%s) (%s) {\n", mockName, m.name, paramDecls(m.params), resultDecls(m.results))
		buf.WriteString("\tm.mu.Lock()\n")
		fmt.Fprintf(&buf, "\tm.%sCalls = append(m.%sCalls, %s_%sCall{%s %s})\n", m.name, m.name, name, m.name, valuesList("Arg", m.params), valuesList("Ret", nil))
		buf.WriteString("\tstub := m.")
		buf.WriteString(m.name)
		buf.WriteString("Stub\n\tm.mu.Unlock()\n")
		buf.WriteString("\tif stub != nil { return stub(" + namesList(m.params) + ") }\n")
		// return zero values when no stub
		if len(m.results) == 0 {
			buf.WriteString("\treturn\n")
		} else {
			buf.WriteString("\treturn " + zeroValuesList(m.results) + "\n")
		}
		buf.WriteString("}\n\n")
	}

	// Reset helper
	fmt.Fprintf(&buf, "func (m *%s) Reset() {\n\tm.mu.Lock()\n", mockName)
	for _, m := range methods {
		fmt.Fprintf(&buf, "\tm.%sStub = nil\n\tm.%sCalls = nil\n", m.name, m.name)
	}
	buf.WriteString("\tm.mu.Unlock()\n}\n")

	// gofmt
	fmted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted for easier debugging
		return buf.String(), nil
	}
	return string(fmted), nil
}

type method struct {
	name    string
	params  []types.Type
	results []types.Type
}

func collectMethods(iface *types.Interface) []method {
	var ms []method
	for i := 0; i < iface.NumMethods(); i++ {
		m := iface.Method(i)
		sig := m.Type().(*types.Signature)
		params := tupleTypes(sig.Params())
		results := tupleTypes(sig.Results())
		ms = append(ms, method{name: m.Name(), params: params, results: results})
	}
	// Stable ordering
	sort.Slice(ms, func(i, j int) bool { return ms[i].name < ms[j].name })
	return ms
}

func tupleTypes(t *types.Tuple) []types.Type {
	if t == nil {
		return nil
	}
	out := make([]types.Type, t.Len())
	for i := 0; i < t.Len(); i++ {
		out[i] = t.At(i).Type()
	}
	return out
}

func joinFieldList(ts []types.Type) string {
	if len(ts) == 0 {
		return ""
	}
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = types.TypeString(t, qualifier)
	}
	return strings.Join(parts, ", ")
}

func paramDecls(ts []types.Type) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = fmt.Sprintf("a%d %s", i, types.TypeString(t, qualifier))
	}
	return strings.Join(parts, ", ")
}

func resultDecls(ts []types.Type) string { return joinFieldList(ts) }

func fieldsDecl(prefix string, ts []types.Type) string {
	if len(ts) == 0 {
		return ""
	}
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = fmt.Sprintf("%s%d %s", prefix, i, types.TypeString(t, qualifier))
	}
	return strings.Join(parts, " ")
}

func valuesList(prefix string, ts []types.Type) string {
	if len(ts) == 0 {
		return ""
	}
	parts := make([]string, len(ts))
	for i := range ts {
		parts[i] = fmt.Sprintf("%s%d: a%d,", prefix, i, i)
	}
	return strings.Join(parts, " ")
}

func namesList(ts []types.Type) string {
	parts := make([]string, len(ts))
	for i := range ts {
		parts[i] = fmt.Sprintf("a%d", i)
	}
	return strings.Join(parts, ", ")
}

func zeroValuesList(ts []types.Type) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = zeroValue(t)
	}
	return strings.Join(parts, ", ")
}

func zeroValue(t types.Type) string {
	switch ut := t.Underlying().(type) {
	case *types.Basic:
		switch ut.Kind() {
		case types.Bool:
			return "false"
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64, types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr, types.Float32, types.Float64, types.Complex64, types.Complex128:
			return "0"
		case types.String:
			return "\"\""
		default:
			return "nil"
		}
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return "nil"
	case *types.Array:
		return fmt.Sprintf("%s{}", types.TypeString(t, qualifier))
	case *types.Struct:
		return fmt.Sprintf("%s{}", types.TypeString(t, qualifier))
	default:
		return "nil"
	}
}

func qualifier(p *types.Package) string {
	if p == nil {
		return ""
	}
	return p.Name()
}

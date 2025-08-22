package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/orizon-lang/orizon/internal/cli"
)

func main() {
	var (
		showVersion    = flag.Bool("version", false, "show version information")
		showHelp       = flag.Bool("help", false, "show help information")
		jsonOutput     = flag.Bool("json", false, "output version in JSON format")
		outputDir      = flag.String("output", "docs", "output directory for documentation")
		format         = flag.String("format", "html", "output format: html, markdown, json")
		includePrivate = flag.Bool("private", false, "include private symbols")
		packagePath    = flag.String("package", ".", "package path to document")
		title          = flag.String("title", "Orizon Documentation", "documentation title")
		verbose        = flag.Bool("verbose", false, "verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [PACKAGES...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Orizon documentation generator.\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s                          # Document current package\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --format markdown        # Generate Markdown docs\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --output ./api-docs      # Custom output directory\n", os.Args[0])
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		cli.PrintVersion("Orizon Documentation Generator", *jsonOutput)
		os.Exit(0)
	}

	packages := flag.Args()
	if len(packages) == 0 {
		packages = []string{*packagePath}
	}

	generator := &DocGenerator{
		OutputDir:      *outputDir,
		Format:         *format,
		IncludePrivate: *includePrivate,
		Title:          *title,
		Verbose:        *verbose,
	}

	for _, pkg := range packages {
		if err := generator.GeneratePackageDocs(pkg); err != nil {
			cli.ExitWithError("failed to generate documentation for package %s: %v", pkg, err)
		}
	}

	fmt.Printf("Documentation generated in %s\n", *outputDir)
}

type DocGenerator struct {
	OutputDir      string
	Format         string
	IncludePrivate bool
	Title          string
	Verbose        bool
}

type PackageDoc struct {
	Name      string    `json:"name"`
	Synopsis  string    `json:"synopsis"`
	Doc       string    `json:"doc"`
	Functions []FuncDoc `json:"functions"`
	Types     []TypeDoc `json:"types"`
}

type FuncDoc struct {
	Name      string `json:"name"`
	Doc       string `json:"doc"`
	Signature string `json:"signature"`
}

type TypeDoc struct {
	Name    string    `json:"name"`
	Doc     string    `json:"doc"`
	Kind    string    `json:"kind"`
	Methods []FuncDoc `json:"methods"`
}

func (g *DocGenerator) GeneratePackageDocs(packagePath string) error {
	if g.Verbose {
		fmt.Printf("Generating documentation for package: %s\n", packagePath)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, packagePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	for pkgName, pkg := range pkgs {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		docPkg := doc.New(pkg, packagePath, doc.AllDecls)
		packageDoc := g.convertPackageDoc(docPkg)

		if err := g.writePackageDoc(packageDoc); err != nil {
			return fmt.Errorf("failed to write documentation: %w", err)
		}
	}

	return nil
}

func (g *DocGenerator) convertPackageDoc(docPkg *doc.Package) *PackageDoc {
	pkg := &PackageDoc{
		Name:     docPkg.Name,
		Synopsis: docPkg.Synopsis(docPkg.Doc),
		Doc:      docPkg.Doc,
	}

	for _, fn := range docPkg.Funcs {
		if !g.IncludePrivate && !ast.IsExported(fn.Name) {
			continue
		}

		funcDoc := FuncDoc{
			Name:      fn.Name,
			Doc:       fn.Doc,
			Signature: fmt.Sprintf("func %s(...)", fn.Name),
		}

		pkg.Functions = append(pkg.Functions, funcDoc)
	}

	for _, typ := range docPkg.Types {
		if !g.IncludePrivate && !ast.IsExported(typ.Name) {
			continue
		}

		typeDoc := TypeDoc{
			Name: typ.Name,
			Doc:  typ.Doc,
			Kind: "type",
		}

		for _, method := range typ.Methods {
			if !g.IncludePrivate && !ast.IsExported(method.Name) {
				continue
			}

			methodDoc := FuncDoc{
				Name:      method.Name,
				Doc:       method.Doc,
				Signature: fmt.Sprintf("func %s(...)", method.Name),
			}

			typeDoc.Methods = append(typeDoc.Methods, methodDoc)
		}

		pkg.Types = append(pkg.Types, typeDoc)
	}

	return pkg
}

func (g *DocGenerator) writePackageDoc(pkg *PackageDoc) error {
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	switch g.Format {
	case "json":
		return g.writeJSON(pkg)
	case "markdown":
		return g.writeMarkdown(pkg)
	case "html":
		return g.writeHTML(pkg)
	default:
		return fmt.Errorf("unsupported format: %s", g.Format)
	}
}

func (g *DocGenerator) writeJSON(pkg *PackageDoc) error {
	filename := filepath.Join(g.OutputDir, pkg.Name+".json")
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (g *DocGenerator) writeMarkdown(pkg *PackageDoc) error {
	filename := filepath.Join(g.OutputDir, pkg.Name+".md")

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# Package %s\n\n", pkg.Name))

	if pkg.Synopsis != "" {
		buf.WriteString(fmt.Sprintf("%s\n\n", pkg.Synopsis))
	}

	if pkg.Doc != "" {
		buf.WriteString(fmt.Sprintf("%s\n\n", pkg.Doc))
	}

	if len(pkg.Functions) > 0 {
		buf.WriteString("## Functions\n\n")
		for _, fn := range pkg.Functions {
			buf.WriteString(fmt.Sprintf("### %s\n\n", fn.Name))
			buf.WriteString(fmt.Sprintf("```go\n%s\n```\n\n", fn.Signature))
			if fn.Doc != "" {
				buf.WriteString(fmt.Sprintf("%s\n\n", fn.Doc))
			}
		}
	}

	return os.WriteFile(filename, []byte(buf.String()), 0644)
}

func (g *DocGenerator) writeHTML(pkg *PackageDoc) error {
	tmplContent := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - {{.Name}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        .package-name { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
        .function { background: #f8f9fa; padding: 15px; margin: 10px 0; border-left: 4px solid #3498db; }
        .signature { background: #2c3e50; color: white; padding: 10px; font-family: monospace; }
    </style>
</head>
<body>
    <h1 class="package-name">Package {{.Name}}</h1>
    <h2>Functions</h2>
    {{range .Functions}}
    <div class="function">
        <h3>{{.Name}}</h3>
        <div class="signature">{{.Signature}}</div>
        <p>{{.Doc}}</p>
    </div>
    {{end}}
</body>
</html>`

	tmpl, err := template.New("package").Parse(tmplContent)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.OutputDir, pkg.Name+".html")
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		*PackageDoc
		Title string
	}{
		PackageDoc: pkg,
		Title:      g.Title,
	}

	return tmpl.Execute(file, data)
}

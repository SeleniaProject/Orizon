package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/orizon-lang/orizon/internal/testrunner/mockgen"
)

type locale struct {
	okGenerated func(dest string) string
	errMsg      func(msg string) string
	usage       string
}

func getLocale(lang string) locale {
	switch strings.ToLower(lang) {
	case "ja", "jp", "japanese":
		return locale{
			okGenerated: func(dest string) string { return "モックを生成しました: " + dest },
			errMsg:      func(msg string) string { return "エラー: " + msg },
			usage:       "使用方法: orizon-mockgen -interface <名前> [-pkg <生成パッケージ>] [-out <出力先>] [-source <パターン,カンマ区切り>] [-tags <ビルドタグ,カンマ区切り>] [-lang ja|en]",
		}
	default:
		return locale{
			okGenerated: func(dest string) string { return "Mock generated: " + dest },
			errMsg:      func(msg string) string { return "Error: " + msg },
			usage:       "Usage: orizon-mockgen -interface <name> [-pkg <generated package>] [-out <destination>] [-source <patterns,comma-separated>] [-tags <build-tags,comma-separated>] [-lang ja|en]",
		}
	}
}

func main() {
	var (
		iface   string
		genPkg  string
		out     string
		sources string
		tags    string
		lang    string
	)
	flag.StringVar(&iface, "interface", "", "interface name to mock (required)")
	flag.StringVar(&genPkg, "pkg", "", "generated package name (default: <src pkg>mock)")
	flag.StringVar(&out, "out", "", "destination file path (writes to file when set)")
	flag.StringVar(&sources, "source", "./...", "source package patterns (comma-separated)")
	flag.StringVar(&tags, "tags", "", "build tags (comma-separated)")
	flag.StringVar(&lang, "lang", "en", "message language (ja|en)")
	flag.Parse()

	L := getLocale(lang)

	if strings.TrimSpace(iface) == "" {
		fmt.Fprintln(os.Stderr, L.errMsg("-interface is required"))
		fmt.Fprintln(os.Stderr, L.usage)
		os.Exit(2)
	}
	var src []string
	for _, p := range strings.Split(sources, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			src = append(src, p)
		}
	}
	var tagSlice []string
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tagSlice = append(tagSlice, t)
		}
	}

	code, err := mockgen.Generate(mockgen.GenOptions{
		InterfaceName:  iface,
		PackageName:    genPkg,
		Destination:    out,
		SourcePatterns: src,
		BuildTags:      tagSlice,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, L.errMsg(err.Error()))
		os.Exit(1)
	}

	if out != "" {
		fmt.Fprintln(os.Stdout, L.okGenerated(out))
		return
	}
	fmt.Fprintln(os.Stdout, code)
}

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/loopcontext/msgcat"
)

// extractConfig holds flags for the extract command.
type extractConfig struct {
	paths         []string
	out           string
	source        string
	format        string
	includeTests  bool
	msgcatPkg     string
	excludeDirs   string
}

func usageExtract() {
	fmt.Fprintf(os.Stderr, `usage: msgcat extract [options] [paths]

Extract discovers message keys referenced in Go code (GetMessageWithCtx, WrapErrorWithCtx,
GetErrorWithCtx) and optionally syncs them into a source language YAML file.

If no paths are provided, scans the current directory.

Modes:
  - Keys only: omit -source; writes unique keys (one per line) to -out or stdout.
  - Sync to YAML: set -source to a msgcat YAML file; adds missing keys with empty short/long, writes to -out.

Flags:
`)
	flag.CommandLine.PrintDefaults()
}

func parseExtractFlags(args []string) (*extractConfig, error) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	fs.Usage = usageExtract
	var cfg extractConfig
	fs.StringVar(&cfg.out, "out", "", "Output file (keys: one key per line; sync: YAML path). Default stdout for keys.")
	fs.StringVar(&cfg.source, "source", "", "Source language YAML path (enables sync mode).")
	fs.StringVar(&cfg.format, "format", "keys", "For keys: 'keys' (one per line) or 'yaml'. For sync: ignored.")
	fs.BoolVar(&cfg.includeTests, "include-tests", false, "Include _test.go files.")
	fs.StringVar(&cfg.msgcatPkg, "msgcat-pkg", "github.com/loopcontext/msgcat", "Import path for msgcat (detect calls from this package).")
	fs.StringVar(&cfg.excludeDirs, "exclude", "vendor", "Comma-separated dir names to skip (e.g. vendor).")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	cfg.paths = fs.Args()
	if len(cfg.paths) == 0 {
		cfg.paths = []string{"."}
	}
	return &cfg, nil
}

// keyExtractor collects message keys from Go files via AST and MessageDef struct literals.
type keyExtractor struct {
	msgcatImport string
	msgcatName   string // local name in current file (e.g. "msgcat")
	keys         map[string]struct{}
	defs         map[string]msgcat.RawMessage // key -> content from MessageDef literals
	methodArgIdx map[string]int
}

func newKeyExtractor(msgcatImport string) *keyExtractor {
	return &keyExtractor{
		msgcatImport: msgcatImport,
		keys:         make(map[string]struct{}),
		defs:         make(map[string]msgcat.RawMessage),
		methodArgIdx: map[string]int{
			"GetMessageWithCtx":  1,
			"GetErrorWithCtx":   1,
			"WrapErrorWithCtx":  2,
		},
	}
}

func (e *keyExtractor) extractFromFile(path string, src []byte) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return err
	}
	e.msgcatName = e.msgcatImportName(f)
	if e.msgcatName == "" {
		return nil
	}
	ast.Walk(e, f)
	return nil
}

func (e *keyExtractor) msgcatImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, `"`)
		if path != e.msgcatImport {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "msgcat"
	}
	return ""
}

func (e *keyExtractor) Visit(node ast.Node) ast.Visitor {
	// MessageDef struct literals (standalone, or in slice/map)
	if cl, ok := node.(*ast.CompositeLit); ok {
		e.visitCompositeLit(cl)
		return e
	}
	// API calls: GetMessageWithCtx, WrapErrorWithCtx, GetErrorWithCtx
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return e
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return e
	}
	idx, ok := e.methodArgIdx[sel.Sel.Name]
	if !ok {
		return e
	}
	if idx >= len(call.Args) {
		return e
	}
	key := e.extractString(call.Args[idx])
	if key != "" {
		e.keys[key] = struct{}{}
	}
	return e
}

func (e *keyExtractor) visitCompositeLit(cl *ast.CompositeLit) {
	switch t := cl.Type.(type) {
	case *ast.SelectorExpr:
		if e.isMessageDefType(t) {
			key, raw := e.extractMessageDefFromCompositeLit(cl)
			if key != "" {
				e.defs[key] = raw
				e.keys[key] = struct{}{}
			}
		}
	case *ast.ArrayType:
		if e.isMessageDefType(t.Elt) {
			for _, elt := range cl.Elts {
				if inner, ok := elt.(*ast.CompositeLit); ok {
					key, raw := e.extractMessageDefFromCompositeLit(inner)
					if key != "" {
						e.defs[key] = raw
						e.keys[key] = struct{}{}
					}
				}
			}
		}
	case *ast.MapType:
		if e.isMessageDefType(t.Value) {
			for _, elt := range cl.Elts {
				kve, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				inner, ok := kve.Value.(*ast.CompositeLit)
				if !ok {
					continue
				}
				key, raw := e.extractMessageDefFromCompositeLit(inner)
				if key != "" {
					e.defs[key] = raw
					e.keys[key] = struct{}{}
				}
			}
		}
	}
}

func (e *keyExtractor) extractString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.BasicLit:
		if t.Kind == token.STRING {
			s, _ := unquote(t.Value)
			return s
		}
	case *ast.BinaryExpr:
		if t.Op == token.ADD {
			return e.extractString(t.X) + e.extractString(t.Y)
		}
	}
	return ""
}

func unquote(s string) (string, error) {
	if len(s) < 2 || s[0] != '"' {
		return s, nil
	}
	// Simple unquote: strip quotes and handle \"
	var b strings.Builder
	for i := 1; i < len(s)-1; i++ {
		if s[i] == '\\' && i+1 < len(s)-1 {
			i++
			if s[i] == '"' {
				b.WriteByte('"')
			} else {
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String(), nil
}

func (e *keyExtractor) sortedKeys() []string {
	out := make([]string, 0, len(e.keys))
	for k := range e.keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func runExtract(cfg *extractConfig) error {
	excludeSet := make(map[string]struct{})
	for _, d := range strings.Split(cfg.excludeDirs, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			excludeSet[d] = struct{}{}
		}
	}
	ext := newKeyExtractor(cfg.msgcatPkg)
	for _, path := range cfg.paths {
		path = filepath.Clean(path)
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					if _, skip := excludeSet[info.Name()]; skip {
						return filepath.SkipDir
					}
					return nil
				}
				if filepath.Ext(p) != ".go" {
					return nil
				}
				if !cfg.includeTests && strings.HasSuffix(p, "_test.go") {
					return nil
				}
				src, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				return ext.extractFromFile(p, src)
			})
		} else {
			if filepath.Ext(path) != ".go" {
				continue
			}
			if !cfg.includeTests && strings.HasSuffix(path, "_test.go") {
				continue
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = ext.extractFromFile(path, src)
		}
		if err != nil {
			return err
		}
	}
	keys := ext.sortedKeys()
	if cfg.source != "" {
		return runExtractSync(cfg, keys, ext.defs)
	}
	// Keys-only output (one per line)
	out := strings.Join(keys, "\n")
	if out != "" {
		out += "\n"
	}
	if cfg.out != "" {
		return os.WriteFile(cfg.out, []byte(out), 0644)
	}
	fmt.Print(out)
	return nil
}

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// coderPackages enumerates dodder's versioned-coder packages — the ones
// that hold a `map[string]coder` keyed by `!type-string` and call
// registerTommy / registerBuiltinTypeString. The analyzer scans each of
// these and cross-references against internal/bravo/ids/types_builtin.go.
//
// This list is curated, not auto-discovered — there's no naming
// convention that reliably distinguishes "versioned-coder package"
// from other `*_blobs`/`*_configs`/`*_stores` packages (e.g.
// lib/charlie/config_cli is unrelated). When you add, remove, or move
// a versioned-coder package, update this list.
var coderPackages = []string{
	"internal/charlie/repo_blobs",
	"internal/delta/repo_configs",
	"internal/charlie/genesis_configs",
	"internal/delta/zettel_id_log",
	"internal/echo/workspace_config_blobs",
	"internal/golf/type_blobs",
}

func main() {
	root := findGoRoot()

	constants, aliases, registrations, err := parseTypesBuiltin(
		filepath.Join(root, "internal/bravo/ids/types_builtin.go"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing types_builtin.go: %v\n", err)
		os.Exit(1)
	}

	var coderKeys []coderEntry

	for _, pkg := range coderPackages {
		dir := filepath.Join(root, pkg)
		entries, err := parseCoderPackage(dir, constants)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", pkg, err)
			os.Exit(1)
		}
		coderKeys = append(coderKeys, entries...)
	}

	exitCode := 0
	report := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		exitCode = 1
	}

	// Check 1: Every coder map key has a matching constant.
	for _, entry := range coderKeys {
		if _, ok := constants[entry.typeString]; !ok {
			// The coder used an ids.Type* identifier — check if it resolved.
			if entry.typeString == "" {
				report(
					"%s: coder map key %q could not be resolved to a type string",
					entry.file, entry.rawKey,
				)
			} else {
				report(
					"%s: coder map key %q has no matching constant in types_builtin.go",
					entry.file, entry.typeString,
				)
			}
		}
	}

	// Check 2: Every coder map key has a registerBuiltinTypeString call.
	for _, entry := range coderKeys {
		if entry.typeString == "" {
			continue
		}
		if _, ok := registrations[entry.typeString]; !ok {
			report(
				"%s: coder map key %q is not registered via registerBuiltinTypeString",
				entry.file, entry.typeString,
			)
		}
	}

	// Check 3: Every registerBuiltinTypeString call references a defined constant.
	for typeString, reg := range registrations {
		if _, ok := constants[typeString]; !ok {
			report(
				"types_builtin.go: registerBuiltinTypeString(%q) references undefined constant (identifier: %s)",
				typeString, reg.identName,
			)
		}
	}

	// Check 4: At most one default per genre.
	genreDefaults := make(map[string][]string) // genre -> []typeString
	for _, reg := range registrations {
		if reg.isDefault {
			genreDefaults[reg.genre] = append(genreDefaults[reg.genre], reg.typeString)
		}
	}
	for genre, types := range genreDefaults {
		if len(types) > 1 {
			report(
				"types_builtin.go: genre %q has multiple defaults: %s",
				genre, strings.Join(types, ", "),
			)
		}
	}

	// Check 5: VCurrent aliases point to a type registered as default.
	for aliasName, targetIdent := range aliases {
		targetValue, ok := constants[aliasName]
		if !ok {
			// The alias itself — skip, it's the alias name not a type string.
			// We need to check the resolved value.
			_ = targetIdent
			continue
		}
		_ = targetValue
	}

	// Better approach for VCurrent: check that aliases resolve to registered defaults.
	for aliasName, targetIdent := range aliases {
		if !strings.HasSuffix(aliasName, "VCurrent") {
			continue
		}
		// Resolve the target identifier to its string value.
		targetTypeString := ""
		for ts, constInfo := range constants {
			if constInfo == targetIdent {
				targetTypeString = ts
				break
			}
		}
		if targetTypeString == "" {
			// Try resolving targetIdent as a constant name.
			for ts, ci := range constants {
				_ = ci
				if ts == targetIdent {
					targetTypeString = ts
					break
				}
			}
		}

		// Find the registration for the target.
		// Skip types with genre Unknown — they use VCurrent as a convenience
		// alias without participating in the Default() mechanism.
		if targetTypeString != "" {
			if reg, ok := registrations[targetTypeString]; ok {
				if !reg.isDefault && reg.genre != "Unknown" {
					report(
						"types_builtin.go: VCurrent alias %s points to %q which is not marked as default",
						aliasName, targetTypeString,
					)
				}
			}
		}
	}

	os.Exit(exitCode)
}

// findGoRoot walks up from cwd to find the directory containing go.mod.
func findGoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot get working directory: %v\n", err)
		os.Exit(1)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintf(os.Stderr, "cannot find go.mod in any parent directory\n")
			os.Exit(1)
		}
		dir = parent
	}
}

type registration struct {
	typeString string
	identName  string
	genre      string
	isDefault  bool
}

// constants maps type string value -> constant identifier name.
// aliases maps constant identifier name -> target identifier name (for VCurrent = TypeFooV1).
// registrations maps type string value -> registration info.
func parseTypesBuiltin(
	path string,
) (
	constants map[string]string,
	aliases map[string]string,
	registrations map[string]registration,
	err error,
) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	constants = make(map[string]string)
	aliases = make(map[string]string)
	registrations = make(map[string]registration)

	// Pass 1: Extract constants.
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			name := valueSpec.Names[0].Name

			switch v := valueSpec.Values[0].(type) {
			case *ast.BasicLit:
				if v.Kind == token.STRING {
					s, err := strconv.Unquote(v.Value)
					if err != nil {
						continue
					}
					if strings.HasPrefix(s, "!") {
						constants[s] = name
					}
				}
			case *ast.Ident:
				// Alias: TypeFooVCurrent = TypeFooV1
				aliases[name] = v.Name
			}
		}
	}

	// Pass 2: Extract registerBuiltinTypeString calls from init().
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != "init" {
			continue
		}
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			ident, ok := callExpr.Fun.(*ast.Ident)
			if !ok || ident.Name != "registerBuiltinTypeString" {
				return true
			}
			if len(callExpr.Args) < 3 {
				return true
			}

			reg := registration{}

			// First arg: type constant identifier or string literal.
			switch arg := callExpr.Args[0].(type) {
			case *ast.Ident:
				reg.identName = arg.Name
				// Resolve to string value.
				for ts, constName := range constants {
					if constName == arg.Name {
						reg.typeString = ts
						break
					}
				}
			case *ast.BasicLit:
				if arg.Kind == token.STRING {
					s, _ := strconv.Unquote(arg.Value)
					reg.typeString = s
					reg.identName = s
				}
			}

			// Second arg: genre (genres.Foo selector).
			if sel, ok := callExpr.Args[1].(*ast.SelectorExpr); ok {
				reg.genre = sel.Sel.Name
			}

			// Third arg: isDefault bool.
			if ident, ok := callExpr.Args[2].(*ast.Ident); ok {
				reg.isDefault = ident.Name == "true"
			}

			if reg.typeString != "" {
				registrations[reg.typeString] = reg
			}

			return true
		})
	}

	return constants, aliases, registrations, nil
}

type coderEntry struct {
	typeString string // resolved type string value
	rawKey     string // raw identifier or string as written in source
	file       string // source file path
}

func parseCoderPackage(
	dir string,
	constants map[string]string,
) ([]coderEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var result []coderEntry
	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") ||
			strings.HasSuffix(name, "_test.go") ||
			strings.HasSuffix(name, "_tommy.go") {
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CompositeLit:
				result = append(result, extractMapKeys(node, path, constants)...)
			case *ast.CallExpr:
				if e := extractRegisterTommyKey(node, path, constants); e != nil {
					result = append(result, *e)
				}
			}
			return true
		})
	}

	return result, nil
}

func extractMapKeys(
	lit *ast.CompositeLit,
	file string,
	constants map[string]string,
) []coderEntry {
	// Only match map[string]... composite literals.
	mapType, ok := lit.Type.(*ast.MapType)
	if !ok {
		return nil
	}
	// Check if key type is string.
	keyIdent, ok := mapType.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return nil
	}

	var entries []coderEntry
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if e := resolveTypeRef(kv.Key, file, constants); e != nil {
			entries = append(entries, *e)
		}
	}
	return entries
}

func extractRegisterTommyKey(
	call *ast.CallExpr,
	file string,
	constants map[string]string,
) *coderEntry {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok || ident.Name != "registerTommy" {
		return nil
	}
	// registerTommy(typeMap, typeString, decode, encode)
	// The type string is the second argument.
	if len(call.Args) < 2 {
		return nil
	}
	return resolveTypeRef(call.Args[1], file, constants)
}

func resolveTypeRef(
	expr ast.Expr,
	file string,
	constants map[string]string,
) *coderEntry {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			s, err := strconv.Unquote(v.Value)
			if err != nil {
				return nil
			}
			// Only treat string literals starting with "!" as type strings.
			// Other map[string] keys (e.g. formatter IDs) are not type references.
			if !strings.HasPrefix(s, "!") {
				return nil
			}
			return &coderEntry{typeString: s, rawKey: s, file: file}
		}
	case *ast.SelectorExpr:
		// ids.TypeFoo
		if x, ok := v.X.(*ast.Ident); ok && x.Name == "ids" {
			identName := v.Sel.Name
			for ts, constName := range constants {
				if constName == identName {
					return &coderEntry{
						typeString: ts,
						rawKey:     "ids." + identName,
						file:       file,
					}
				}
			}
			// Identifier not found in constants — unresolved.
			return &coderEntry{
				typeString: "",
				rawKey:     "ids." + identName,
				file:       file,
			}
		}
	}
	return nil
}

// reflexive_interface_generator generates interfaces from concrete types
// Usage: go run . -type=TypeName

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	typeNames = flag.String("type", "", "comma-separated list of type names; must be set")
	output    = flag.String("output", "", "output file name; default srcdir/<type>_interface.go")
	buildTags = flag.String("tags", "", "comma-separated list of build tags to apply")
)

// Usage prints usage message and exits
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\treflexive_interface_generator -type T [options]\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("reflexive_interface_generator: ")
	flag.Usage = Usage
	flag.Parse()

	if len(*typeNames) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	types := strings.Split(*typeNames, ",")
	tags := strings.Split(*buildTags, ",")

	// Load package information
	cfg := &packages.Config{
		Mode:       packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests:      false,
		BuildFlags: buildTagsToFlags(tags),
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		log.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	if len(pkgs) == 0 {
		log.Fatal("no packages found")
	}

	pkg := pkgs[0]

	// Process each type
	for _, typeName := range types {
		typeName = strings.TrimSpace(typeName)
		if err := generateInterface(pkg, typeName); err != nil {
			log.Fatal(err)
		}
	}
}

// buildTagsToFlags converts build tags to compiler flags
func buildTagsToFlags(tags []string) []string {
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		return nil
	}
	return []string{"-tags=" + strings.Join(tags, ",")}
}

// generateInterface generates an interface for the given type
func generateInterface(pkg *packages.Package, typeName string) error {
	// Find the type in the package
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return fmt.Errorf("type %s not found in package %s", typeName, pkg.Name)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return fmt.Errorf("%s is not a named type", typeName)
	}

	// Collect methods
	methods := collectMethods(named)
	if len(methods) == 0 {
		return fmt.Errorf("type %s has no exported methods", typeName)
	}

	// Generate interface code
	interfaceName := "I" + typeName
	code := generateInterfaceCode(pkg.Name, interfaceName, methods, typeName)

	// Determine output file
	outputFile := *output
	if outputFile == "" {
		outputFile = strings.ToLower(typeName) + "_interface.go"
	}

	// Write the file
	if err := writeFile(outputFile, code); err != nil {
		return err
	}

	// Run goimports on the generated file to fix imports
	if err := runGoimports(outputFile); err != nil {
		log.Printf("Warning: goimports failed: %v", err)
		// Continue anyway - file is still valid Go code
	}

	log.Printf("Generated interface %s for type %s in %s", interfaceName, typeName, outputFile)
	return nil
}

// Method represents a method signature
type Method struct {
	Name      string
	Signature string
	Comments  []string
}

// collectMethods collects all exported methods from a type
func collectMethods(named *types.Named) []Method {
	var methods []Method

	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)

		// Only include exported methods
		if !method.Exported() {
			continue
		}

		sig := method.Type().(*types.Signature)

		// Format the method signature
		methodSig := formatMethodSignature(method.Name(), sig)

		methods = append(methods, Method{
			Name:      method.Name(),
			Signature: methodSig,
		})
	}

	return methods
}

// formatMethodSignature formats a method signature for the interface
func formatMethodSignature(name string, sig *types.Signature) string {
	var buf bytes.Buffer

	buf.WriteString(name)

	// Format parameters
	buf.WriteString("(")
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		param := params.At(i)
		if param.Name() != "" {
			buf.WriteString(param.Name())
			buf.WriteString(" ")
		}
		buf.WriteString(types.TypeString(param.Type(), nil))
	}
	buf.WriteString(")")

	// Format results
	results := sig.Results()
	if results.Len() > 0 {
		buf.WriteString(" ")
		if results.Len() == 1 && results.At(0).Name() == "" {
			buf.WriteString(types.TypeString(results.At(0).Type(), nil))
		} else {
			buf.WriteString("(")
			for i := 0; i < results.Len(); i++ {
				if i > 0 {
					buf.WriteString(", ")
				}
				result := results.At(i)
				if result.Name() != "" {
					buf.WriteString(result.Name())
					buf.WriteString(" ")
				}
				buf.WriteString(types.TypeString(result.Type(), nil))
			}
			buf.WriteString(")")
		}
	}

	return buf.String()
}

// generateInterfaceCode generates the Go code for the interface
func generateInterfaceCode(pkgName, interfaceName string, methods []Method, originalType string) []byte {
	var buf bytes.Buffer

	// Write header
	fmt.Fprintf(&buf, "// Code generated by reflexive_interface_generator -type=%s; DO NOT EDIT.\n\n", originalType)
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)

	// No imports section - goimports will handle this automatically

	// Write interface
	fmt.Fprintf(&buf, "// %s is an interface that mirrors all methods of %s.\n", interfaceName, originalType)
	fmt.Fprintf(&buf, "type %s interface {\n", interfaceName)

	for _, method := range methods {
		fmt.Fprintf(&buf, "\t%s\n", method.Signature)
	}

	fmt.Fprintf(&buf, "}\n\n")

	// Add compile-time check that the original type implements the interface
	fmt.Fprintf(&buf, "// Compile-time check that %s implements %s.\n", originalType, interfaceName)
	fmt.Fprintf(&buf, "var _ %s = (*%s)(nil)\n", interfaceName, originalType)

	return buf.Bytes()
}

// runGoimports runs goimports on the specified file to fix imports
func runGoimports(filename string) error {
	cmd := exec.Command("goimports", "-w", filename)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("goimports failed: %v\nOutput: %s", err, output)
	}
	return nil
}

// writeFile writes the generated code to a file
func writeFile(filename string, data []byte) error {
	// Format the code
	formatted, err := format.Source(data)
	if err != nil {
		// If formatting fails, write the unformatted code for debugging
		log.Printf("Warning: gofmt failed: %v", err)
		formatted = data
	}

	return os.WriteFile(filename, formatted, 0o644)
}

// parseFile parses a Go source file to extract method comments
func parseFile(filename string) (*ast.File, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// findTypeFile finds the file containing the type definition
func findTypeFile(pkg *packages.Package, typeName string) (string, error) {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				if typeSpec.Name.Name == typeName {
					pos := pkg.Fset.Position(typeSpec.Pos())
					return pos.Filename, nil
				}
			}
		}
	}

	return "", fmt.Errorf("type %s not found in package files", typeName)
}

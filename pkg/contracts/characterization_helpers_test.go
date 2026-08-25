package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const contractPackageImportPath = "github.com/example/autostream-contracts/pkg/contracts"

type contractPackageModel struct {
	Package *types.Package
	Files   []*ast.File
	Info    *types.Info
}

type publicAPIManifest struct {
	FormatVersion           int                   `json:"format_version"`
	PackageImportPath       string                `json:"package_import_path"`
	PackageName             string                `json:"package_name"`
	ExportedIdentifierCount int                   `json:"exported_identifier_count"`
	ExportedMethodCount     int                   `json:"exported_method_count"`
	Identifiers             []publicAPIIdentifier `json:"identifiers"`
	Methods                 []publicAPIMethod     `json:"methods"`
}

type publicAPIIdentifier struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Type       string `json:"type"`
	Alias      bool   `json:"alias"`
	Underlying string `json:"underlying,omitempty"`
	Value      string `json:"value,omitempty"`
}

type publicAPIMethod struct {
	Receiver  string `json:"receiver"`
	Name      string `json:"name"`
	Signature string `json:"signature"`
}

type structFieldManifest struct {
	FormatVersion int                    `json:"format_version"`
	StructCount   int                    `json:"struct_count"`
	Structs       []exportedStructRecord `json:"structs"`
}

type exportedStructRecord struct {
	Name   string                `json:"name"`
	Alias  bool                  `json:"alias"`
	Fields []exportedFieldRecord `json:"fields"`
}

type exportedFieldRecord struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Exported    bool   `json:"exported"`
	Embedded    bool   `json:"embedded"`
	Type        string `json:"type"`
	StructTag   string `json:"struct_tag"`
	JSONName    string `json:"json_name"`
	JSONIgnored bool   `json:"json_ignored"`
	OmitEmpty   bool   `json:"omitempty"`
}

type enumConstantManifest struct {
	FormatVersion int                  `json:"format_version"`
	ConstantCount int                  `json:"constant_count"`
	Constants     []enumConstantRecord `json:"constants"`
}

type enumConstantRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	EnumType string `json:"enum_type,omitempty"`
	Value    string `json:"value"`
}

type zeroValueWireManifest struct {
	FormatVersion int                   `json:"format_version"`
	StructCount   int                   `json:"struct_count"`
	Structs       []zeroValueWireRecord `json:"structs"`
}

type zeroValueWireRecord struct {
	Name         string `json:"name"`
	WireJSON     string `json:"wire_json,omitempty"`
	WireSHA256   string `json:"wire_sha256,omitempty"`
	Shape        any    `json:"shape,omitempty"`
	MarshalError string `json:"marshal_error,omitempty"`
}

func TestPublicAPICharacterization(t *testing.T) {
	model := loadContractPackageModel(t)
	api, structs, constants := buildPublicAPICharacterization(t, model)

	assertOrCaptureCharacterization(t, "public-api.json", api)
	assertOrCaptureCharacterization(t, "struct-fields.json", structs)
	assertOrCaptureCharacterization(t, "enum-constants.json", constants)
	t.Logf("exported package identifiers: %d; exported methods: %d; exported struct-shaped types: %d",
		api.ExportedIdentifierCount, api.ExportedMethodCount, structs.StructCount)
}

func TestZeroValueWireCharacterization(t *testing.T) {
	model := loadContractPackageModel(t)
	_, structs, _ := buildPublicAPICharacterization(t, model)
	manifest := buildZeroValueWireManifest(t, structs)
	assertOrCaptureCharacterization(t, "zero-value-wire.json", manifest)
}

func TestPublicAPICharacterizationIgnoresSourceFileNamesAndComments(t *testing.T) {
	baseModel := loadContractPackageModel(t)
	baseAPI, baseStructs, baseConstants := buildPublicAPICharacterization(t, baseModel)

	relocatedDirectory := t.TempDir()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read contracts package: %v", err)
	}
	index := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		body = append([]byte("// characterization relocation fixture; comments are intentionally ignored\n"), body...)
		name := fmt.Sprintf("relocated_%03d.go", index)
		if err := os.WriteFile(filepath.Join(relocatedDirectory, name), body, 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		index++
	}
	relocatedModel := loadContractPackageModelFromDirectory(t, relocatedDirectory)
	relocatedAPI, relocatedStructs, relocatedConstants := buildPublicAPICharacterization(t, relocatedModel)
	if !reflect.DeepEqual(baseAPI, relocatedAPI) || !reflect.DeepEqual(baseStructs, relocatedStructs) ||
		!reflect.DeepEqual(baseConstants, relocatedConstants) {
		t.Fatal("public API characterization changed after source-only filename/comment relocation")
	}
}

func loadContractPackageModel(t *testing.T) contractPackageModel {
	t.Helper()
	return loadContractPackageModelFromDirectory(t, ".")
}

func loadContractPackageModelFromDirectory(t *testing.T, directory string) contractPackageModel {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read contracts package: %v", err)
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("no production Go files found in package contracts")
	}

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(names))
	for _, name := range names {
		file, err := parser.ParseFile(fset, filepath.Join(directory, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, file)
	}

	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
	}
	var typeErrors []string
	config := &types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			typeErrors = append(typeErrors, err.Error())
		},
	}
	pkg, err := config.Check(contractPackageImportPath, fset, files, info)
	if err != nil {
		t.Fatalf("type-check package contracts: %v\n%s", err, strings.Join(typeErrors, "\n"))
	}
	if pkg.Name() != "contracts" {
		t.Fatalf("package name=%q, want contracts", pkg.Name())
	}

	return contractPackageModel{Package: pkg, Files: files, Info: info}
}

func buildPublicAPICharacterization(t *testing.T, model contractPackageModel) (publicAPIManifest, structFieldManifest, enumConstantManifest) {
	t.Helper()

	qualifier := func(pkg *types.Package) string {
		if pkg == nil || pkg.Path() == model.Package.Path() {
			return ""
		}
		return pkg.Path()
	}
	typeString := func(typ types.Type) string {
		return types.TypeString(typ, qualifier)
	}

	scope := model.Package.Scope()
	names := scope.Names()
	sort.Strings(names)
	api := publicAPIManifest{
		FormatVersion:     1,
		PackageImportPath: model.Package.Path(),
		PackageName:       model.Package.Name(),
	}
	structManifest := structFieldManifest{FormatVersion: 1}
	constantManifest := enumConstantManifest{FormatVersion: 1}

	for _, name := range names {
		if !ast.IsExported(name) {
			continue
		}
		obj := scope.Lookup(name)
		entry := publicAPIIdentifier{Name: name, Type: typeString(obj.Type())}
		switch typed := obj.(type) {
		case *types.TypeName:
			entry.Kind = "type"
			entry.Alias = typed.IsAlias()
			entry.Underlying = typeString(types.Unalias(typed.Type()).Underlying())
			if structType, ok := types.Unalias(typed.Type()).Underlying().(*types.Struct); ok {
				record := exportedStructRecord{Name: name, Alias: typed.IsAlias()}
				for i := 0; i < structType.NumFields(); i++ {
					field := structType.Field(i)
					if !field.Exported() && !field.Embedded() {
						continue
					}
					tag := structType.Tag(i)
					jsonName, ignored, omitEmpty := characterizeJSONTag(field.Name(), tag)
					record.Fields = append(record.Fields, exportedFieldRecord{
						Index:       i,
						Name:        field.Name(),
						Exported:    field.Exported(),
						Embedded:    field.Embedded(),
						Type:        typeString(field.Type()),
						StructTag:   tag,
						JSONName:    jsonName,
						JSONIgnored: ignored,
						OmitEmpty:   omitEmpty,
					})
				}
				structManifest.Structs = append(structManifest.Structs, record)
			}
		case *types.Const:
			entry.Kind = "const"
			entry.Value = typed.Val().ExactString()
			enumType := ""
			if named, ok := types.Unalias(typed.Type()).(*types.Named); ok && named.Obj().Pkg() == model.Package {
				enumType = named.Obj().Name()
			}
			constantManifest.Constants = append(constantManifest.Constants, enumConstantRecord{
				Name:     name,
				Type:     typeString(typed.Type()),
				EnumType: enumType,
				Value:    typed.Val().ExactString(),
			})
		case *types.Var:
			entry.Kind = "var"
		case *types.Func:
			entry.Kind = "func"
		default:
			t.Fatalf("unsupported exported object %s (%T)", name, obj)
		}
		api.Identifiers = append(api.Identifiers, entry)
	}

	for _, file := range model.Files {
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Recv == nil || !function.Name.IsExported() {
				continue
			}
			obj, ok := model.Info.Defs[function.Name].(*types.Func)
			if !ok {
				t.Fatalf("missing type information for method %s", function.Name.Name)
			}
			signature, ok := obj.Type().(*types.Signature)
			if !ok || signature.Recv() == nil {
				t.Fatalf("invalid method signature for %s", function.Name.Name)
			}
			if !exportedReceiver(signature.Recv().Type()) {
				continue
			}
			api.Methods = append(api.Methods, publicAPIMethod{
				Receiver:  typeString(signature.Recv().Type()),
				Name:      function.Name.Name,
				Signature: typeString(signature),
			})
		}
	}

	sort.Slice(api.Methods, func(i, j int) bool {
		left := api.Methods[i].Receiver + "\x00" + api.Methods[i].Name + "\x00" + api.Methods[i].Signature
		right := api.Methods[j].Receiver + "\x00" + api.Methods[j].Name + "\x00" + api.Methods[j].Signature
		return left < right
	})
	sort.Slice(structManifest.Structs, func(i, j int) bool {
		return structManifest.Structs[i].Name < structManifest.Structs[j].Name
	})
	sort.Slice(constantManifest.Constants, func(i, j int) bool {
		return constantManifest.Constants[i].Name < constantManifest.Constants[j].Name
	})

	api.ExportedIdentifierCount = len(api.Identifiers)
	api.ExportedMethodCount = len(api.Methods)
	structManifest.StructCount = len(structManifest.Structs)
	constantManifest.ConstantCount = len(constantManifest.Constants)
	return api, structManifest, constantManifest
}

func characterizeJSONTag(fieldName, tag string) (name string, ignored, omitEmpty bool) {
	name = fieldName
	raw, ok := reflect.StructTag(tag).Lookup("json")
	if !ok {
		return name, false, false
	}
	parts := strings.Split(raw, ",")
	if parts[0] == "-" {
		return "-", true, false
	}
	if parts[0] != "" {
		name = parts[0]
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			omitEmpty = true
		}
	}
	return name, false, omitEmpty
}

func exportedReceiver(typ types.Type) bool {
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = pointer.Elem()
	}
	named, ok := types.Unalias(typ).(*types.Named)
	return ok && named.Obj().Exported()
}

func buildZeroValueWireManifest(t *testing.T, structs structFieldManifest) zeroValueWireManifest {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	tempDir := t.TempDir()
	goMod := fmt.Sprintf("module autostream-contract-characterization\n\ngo 1.26.5\n\nrequire github.com/example/autostream-contracts v0.0.0\n\nreplace github.com/example/autostream-contracts => %s\n", filepath.ToSlash(repoRoot))
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write temporary go.mod: %v", err)
	}

	var source strings.Builder
	source.WriteString("package main\n\n")
	source.WriteString("import (\n\t\"encoding/json\"\n\t\"os\"\n\n\tcontracts \"github.com/example/autostream-contracts/pkg/contracts\"\n)\n\n")
	source.WriteString("type entry struct { Name string; Value any }\n")
	source.WriteString("type result struct { Name string `json:\"name\"`; JSON json.RawMessage `json:\"json,omitempty\"`; Error string `json:\"error,omitempty\"` }\n\n")
	source.WriteString("func main() {\n\tentries := []entry{\n")
	for _, record := range structs.Structs {
		fmt.Fprintf(&source, "\t\t{Name: %q, Value: contracts.%s{}},\n", record.Name, record.Name)
	}
	source.WriteString("\t}\n\tresults := make([]result, 0, len(entries))\n")
	source.WriteString("\tfor _, item := range entries {\n\t\tbody, err := json.Marshal(item.Value)\n\t\trow := result{Name: item.Name}\n\t\tif err != nil { row.Error = err.Error() } else { row.JSON = body }\n\t\tresults = append(results, row)\n\t}\n")
	source.WriteString("\tencoder := json.NewEncoder(os.Stdout)\n\tencoder.SetEscapeHTML(false)\n\tif err := encoder.Encode(results); err != nil { panic(err) }\n}\n")
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(source.String()), 0o600); err != nil {
		t.Fatalf("write temporary zero-value marshaler: %v", err)
	}

	command := exec.Command("go", "run", "-mod=readonly", ".")
	command.Dir = tempDir
	command.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=auto", "GOMAXPROCS=2")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run temporary zero-value marshaler: %v\n%s", err, output)
	}
	var rows []struct {
		Name  string          `json:"name"`
		JSON  json.RawMessage `json:"json"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(output, &rows); err != nil {
		t.Fatalf("decode zero-value marshaler output: %v\n%s", err, output)
	}
	if len(rows) != len(structs.Structs) {
		t.Fatalf("zero-value marshaler returned %d structs, want %d", len(rows), len(structs.Structs))
	}

	manifest := zeroValueWireManifest{FormatVersion: 1}
	for _, row := range rows {
		record := zeroValueWireRecord{Name: row.Name, MarshalError: row.Error}
		if row.Error == "" {
			record.WireJSON = string(row.JSON)
			digest := sha256.Sum256(row.JSON)
			record.WireSHA256 = hex.EncodeToString(digest[:])
			decoder := json.NewDecoder(bytes.NewReader(row.JSON))
			decoder.UseNumber()
			var value any
			if err := decoder.Decode(&value); err != nil {
				t.Fatalf("decode zero-value JSON for %s: %v", row.Name, err)
			}
			record.Shape = characterizeJSONShape(value)
		}
		manifest.Structs = append(manifest.Structs, record)
	}
	sort.Slice(manifest.Structs, func(i, j int) bool {
		return manifest.Structs[i].Name < manifest.Structs[j].Name
	})
	manifest.StructCount = len(manifest.Structs)
	return manifest
}

func characterizeJSONShape(value any) any {
	switch typed := value.(type) {
	case nil:
		return map[string]any{"kind": "null"}
	case bool:
		return map[string]any{"kind": "boolean"}
	case string:
		return map[string]any{"kind": "string"}
	case json.Number:
		return map[string]any{"kind": "number"}
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, characterizeJSONShape(item))
		}
		return map[string]any{"kind": "array", "items": items}
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fields := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			fields = append(fields, map[string]any{"name": key, "shape": characterizeJSONShape(typed[key])})
		}
		return map[string]any{"kind": "object", "fields": fields}
	default:
		return map[string]any{"kind": fmt.Sprintf("unsupported:%T", value)}
	}
}

func assertOrCaptureCharacterization(t *testing.T, relativePath string, value any) {
	t.Helper()

	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", relativePath, err)
	}
	body = append(body, '\n')
	if bytes.Contains(body, []byte("\r")) {
		t.Fatalf("%s contains a carriage return", relativePath)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	for _, absolute := range []string{repoRoot, filepath.ToSlash(repoRoot)} {
		if absolute != "" && bytes.Contains(bytes.ToLower(body), bytes.ToLower([]byte(absolute))) {
			t.Fatalf("%s contains absolute repository path %q", relativePath, absolute)
		}
	}

	if outputRoot := os.Getenv("AUTOSTREAM_CHARACTERIZATION_OUTPUT"); outputRoot != "" {
		if !filepath.IsAbs(outputRoot) {
			t.Fatalf("AUTOSTREAM_CHARACTERIZATION_OUTPUT must be absolute, got %q", outputRoot)
		}
		outputRoot = filepath.Clean(outputRoot)
		repositoryRoot := filepath.Clean(repoRoot)
		repositoryPrefix := strings.ToLower(repositoryRoot + string(os.PathSeparator))
		if strings.EqualFold(outputRoot, repositoryRoot) || strings.HasPrefix(strings.ToLower(outputRoot), repositoryPrefix) {
			t.Fatalf("AUTOSTREAM_CHARACTERIZATION_OUTPUT must be outside the repository, got %q", outputRoot)
		}
		path := filepath.Clean(filepath.Join(outputRoot, filepath.FromSlash(relativePath)))
		prefix := outputRoot + string(os.PathSeparator)
		if path != outputRoot && !strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix)) {
			t.Fatalf("unsafe characterization output path %q", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create characterization output directory: %v", err)
		}
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatalf("write characterization output %s: %v", path, err)
		}
		return
	}

	goldenPath := filepath.Join("..", "..", "testdata", "characterization", "generated", filepath.FromSlash(relativePath))
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read %s: %v; regenerate with node scripts/characterize-contracts.mjs update", goldenPath, err)
	}
	want = bytes.ReplaceAll(bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n")), []byte("\r"), []byte("\n"))
	if !bytes.Equal(body, want) {
		t.Fatalf("%s drifted; regenerate and review with node scripts/characterize-contracts.mjs update", relativePath)
	}
}

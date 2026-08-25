package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const supportedContractSchemaDraft = "https://json-schema.org/draft/2020-12/schema"

type schemaCharacterizationManifest struct {
	FormatVersion       int                           `json:"format_version"`
	SchemaCount         int                           `json:"schema_count"`
	SupportedDrafts     []string                      `json:"supported_drafts"`
	SupportedExtensions []string                      `json:"supported_extension_keywords"`
	Documents           []schemaCharacterizationDoc   `json:"documents"`
	DuplicateIDs        []schemaCharacterizationIssue `json:"duplicate_ids"`
	InvalidIDs          []schemaCharacterizationIssue `json:"invalid_ids"`
	UnsupportedDrafts   []schemaCharacterizationIssue `json:"unsupported_drafts"`
	UnsupportedKeywords []schemaCharacterizationIssue `json:"unsupported_keywords"`
	MissingRefs         []schemaCharacterizationIssue `json:"missing_refs"`
	ExternalNetworkRefs []schemaCharacterizationIssue `json:"external_network_refs"`
	CompileErrors       []schemaCharacterizationIssue `json:"compile_errors"`
}

type schemaCharacterizationDoc struct {
	Path              string                      `json:"path"`
	ID                string                      `json:"id"`
	Draft             string                      `json:"draft"`
	NormalizedSHA256  string                      `json:"normalized_sha256"`
	Keywords          []string                    `json:"keywords"`
	ExtensionKeywords []string                    `json:"extension_keywords"`
	Refs              []schemaCharacterizationRef `json:"refs"`
	Compiled          bool                        `json:"compiled"`
}

type schemaCharacterizationRef struct {
	Pointer  string `json:"pointer"`
	Value    string `json:"value"`
	Target   string `json:"target"`
	Fragment string `json:"fragment"`
}

type schemaCharacterizationIssue struct {
	Path    string `json:"path"`
	Pointer string `json:"pointer,omitempty"`
	Value   string `json:"value,omitempty"`
	Detail  string `json:"detail"`
}

type loadedContractSchema struct {
	Path        string
	Name        string
	Body        []byte
	Document    any
	Object      map[string]any
	CompilerDoc any
	ID          string
	Draft       string
	Refs        []locatedSchemaRef
	Keywords    map[string]struct{}
	Extensions  map[string]struct{}
}

type locatedSchemaRef struct {
	Pointer  string
	Value    string
	Target   string
	Fragment string
}

type denyContractSchemaURLLoader struct{}

type ecmaContractSchemaRegexp regexp2.Regexp

func (denyContractSchemaURLLoader) Load(location string) (any, error) {
	return nil, fmt.Errorf("external JSON Schema loading is disabled: %s", location)
}

func (expression *ecmaContractSchemaRegexp) MatchString(value string) bool {
	matched, err := (*regexp2.Regexp)(expression).MatchString(value)
	return err == nil && matched
}

func (expression *ecmaContractSchemaRegexp) String() string {
	return (*regexp2.Regexp)(expression).String()
}

func compileECMAContractSchemaRegexp(pattern string) (jsonschema.Regexp, error) {
	expression, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*ecmaContractSchemaRegexp)(expression), nil
}

func TestJSONSchemaCharacterization(t *testing.T) {
	manifest := buildSchemaCharacterizationManifest(t)
	assertOrCaptureCharacterization(t, "schemas.json", manifest)

	issues := len(manifest.DuplicateIDs) + len(manifest.InvalidIDs) + len(manifest.UnsupportedDrafts) +
		len(manifest.UnsupportedKeywords) + len(manifest.MissingRefs) + len(manifest.ExternalNetworkRefs) +
		len(manifest.CompileErrors)
	if issues != 0 {
		t.Fatalf("JSON Schema characterization found %d issue(s); inspect schemas.json", issues)
	}
	if manifest.SchemaCount == 0 {
		t.Fatal("JSON Schema characterization found no schemas")
	}
	t.Logf("parsed, resolved, and compiled %d JSON Schemas without external loading", manifest.SchemaCount)
}

func buildSchemaCharacterizationManifest(t *testing.T) schemaCharacterizationManifest {
	t.Helper()

	paths := discoverContractSchemaPaths(t)
	manifest := schemaCharacterizationManifest{
		FormatVersion:       1,
		SchemaCount:         len(paths),
		SupportedDrafts:     []string{supportedContractSchemaDraft},
		SupportedExtensions: []string{"deprecated", "readOnly", "writeOnly", "x-*"},
	}
	documents := make([]*loadedContractSchema, 0, len(paths))
	pathIndex := make(map[string]*loadedContractSchema, len(paths))
	idIndex := make(map[string]*loadedContractSchema)

	for _, schemaPath := range paths {
		document := loadContractSchemaForCharacterization(t, schemaPath)
		documents = append(documents, document)
		pathIndex[schemaPath] = document
		if document.Draft != supportedContractSchemaDraft {
			manifest.UnsupportedDrafts = append(manifest.UnsupportedDrafts, schemaCharacterizationIssue{
				Path: document.Path, Value: document.Draft, Detail: "unsupported JSON Schema draft",
			})
		}
		if document.ID != "" {
			if detail := validateContractSchemaID(document.ID); detail != "" {
				manifest.InvalidIDs = append(manifest.InvalidIDs, schemaCharacterizationIssue{
					Path: document.Path, Value: document.ID, Detail: detail,
				})
			}
			if previous, exists := idIndex[document.ID]; exists {
				manifest.DuplicateIDs = append(manifest.DuplicateIDs, schemaCharacterizationIssue{
					Path: document.Path, Value: document.ID, Detail: "duplicates " + previous.Path,
				})
			} else {
				idIndex[document.ID] = document
			}
		}
	}

	for _, document := range documents {
		for index := range document.Refs {
			ref := &document.Refs[index]
			target, fragment, classification, detail := resolveContractSchemaRef(document, ref.Value, pathIndex, idIndex)
			ref.Target = target
			ref.Fragment = fragment
			switch classification {
			case "missing":
				manifest.MissingRefs = append(manifest.MissingRefs, schemaCharacterizationIssue{
					Path: document.Path, Pointer: ref.Pointer, Value: ref.Value, Detail: detail,
				})
			case "external":
				manifest.ExternalNetworkRefs = append(manifest.ExternalNetworkRefs, schemaCharacterizationIssue{
					Path: document.Path, Pointer: ref.Pointer, Value: ref.Value, Detail: detail,
				})
			}
			if classification != "ok" {
				continue
			}
			targetDocument := pathIndex[target]
			if targetDocument == nil || !schemaFragmentExists(targetDocument.Document, fragment) {
				manifest.MissingRefs = append(manifest.MissingRefs, schemaCharacterizationIssue{
					Path: document.Path, Pointer: ref.Pointer, Value: ref.Value,
					Detail: fmt.Sprintf("fragment %q does not resolve in %s", fragment, target),
				})
			}
		}
	}

	compileContractSchemas(documents, pathIndex, &manifest)
	for _, document := range documents {
		keywords := sortedStringSet(document.Keywords)
		extensions := sortedStringSet(document.Extensions)
		for _, keyword := range keywords {
			if !supportedContractSchemaKeyword(keyword) {
				manifest.UnsupportedKeywords = append(manifest.UnsupportedKeywords, schemaCharacterizationIssue{
					Path: document.Path, Value: keyword, Detail: "unsupported JSON Schema keyword",
				})
			}
		}
		normalized, err := json.Marshal(document.Document)
		if err != nil {
			t.Fatalf("normalize %s: %v", document.Path, err)
		}
		digest := sha256.Sum256(normalized)
		record := schemaCharacterizationDoc{
			Path:              document.Path,
			ID:                document.ID,
			Draft:             document.Draft,
			NormalizedSHA256:  hex.EncodeToString(digest[:]),
			Keywords:          keywords,
			ExtensionKeywords: extensions,
			Compiled:          true,
		}
		for _, issue := range manifest.CompileErrors {
			if issue.Path == document.Path {
				record.Compiled = false
				break
			}
		}
		for _, ref := range document.Refs {
			record.Refs = append(record.Refs, schemaCharacterizationRef{
				Pointer: ref.Pointer, Value: ref.Value, Target: ref.Target, Fragment: ref.Fragment,
			})
		}
		sort.Slice(record.Refs, func(i, j int) bool {
			left := record.Refs[i].Pointer + "\x00" + record.Refs[i].Value
			right := record.Refs[j].Pointer + "\x00" + record.Refs[j].Value
			return left < right
		})
		manifest.Documents = append(manifest.Documents, record)
	}

	sortSchemaIssues(manifest.DuplicateIDs)
	sortSchemaIssues(manifest.InvalidIDs)
	sortSchemaIssues(manifest.UnsupportedDrafts)
	sortSchemaIssues(manifest.UnsupportedKeywords)
	sortSchemaIssues(manifest.MissingRefs)
	sortSchemaIssues(manifest.ExternalNetworkRefs)
	sortSchemaIssues(manifest.CompileErrors)
	return manifest
}

func discoverContractSchemaPaths(t *testing.T) []string {
	t.Helper()

	root := filepath.Join("..", "..", "schemas")
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			return nil
		}
		relative, err := filepath.Rel(filepath.Join("..", ".."), path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		t.Fatalf("enumerate JSON Schemas: %v", err)
	}
	sort.Strings(paths)
	repositoryRoot := filepath.Join("..", "..")
	if _, err := os.Stat(filepath.Join(repositoryRoot, ".git")); err == nil {
		command := exec.Command("git", "ls-files", "--", "schemas/*.schema.json")
		command.Dir = repositoryRoot
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("enumerate tracked JSON Schemas: %v\n%s", err, output)
		}
		var tracked []string
		for _, line := range strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n") {
			if line != "" {
				tracked = append(tracked, filepath.ToSlash(line))
			}
		}
		sort.Strings(tracked)
		if len(tracked) != len(paths) {
			t.Fatalf("Schema filesystem/tracked inventory differs: filesystem=%d tracked=%d", len(paths), len(tracked))
		}
		for index := range paths {
			if paths[index] != tracked[index] {
				t.Fatalf("Schema filesystem/tracked inventory differs at %d: filesystem=%q tracked=%q", index, paths[index], tracked[index])
			}
		}
	}
	return paths
}

func loadContractSchemaForCharacterization(t *testing.T, schemaPath string) *loadedContractSchema {
	t.Helper()

	osPath := filepath.Join("..", "..", filepath.FromSlash(schemaPath))
	body, err := os.ReadFile(osPath)
	if err != nil {
		t.Fatalf("read %s: %v", schemaPath, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("parse %s: %v", schemaPath, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("parse %s: unexpected trailing JSON content: %v", schemaPath, err)
	}
	object, ok := document.(map[string]any)
	if !ok {
		t.Fatalf("%s root must be a JSON object", schemaPath)
	}
	compilerDocument, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("parse %s with schema compiler: %v", schemaPath, err)
	}
	loaded := &loadedContractSchema{
		Path:        schemaPath,
		Name:        filepath.Base(schemaPath),
		Body:        body,
		Document:    document,
		Object:      object,
		CompilerDoc: compilerDocument,
		Keywords:    make(map[string]struct{}),
		Extensions:  make(map[string]struct{}),
	}
	if value, ok := object["$id"].(string); ok {
		loaded.ID = value
	}
	if value, ok := object["$schema"].(string); ok {
		loaded.Draft = value
	}
	scanContractSchemaObject(object, "", loaded)
	return loaded
}

func scanContractSchemaObject(object map[string]any, pointer string, document *loadedContractSchema) {
	for key, value := range object {
		if strings.HasPrefix(key, "x-") {
			document.Extensions[key] = struct{}{}
		} else {
			document.Keywords[key] = struct{}{}
		}
		if key == "$ref" {
			if ref, ok := value.(string); ok {
				document.Refs = append(document.Refs, locatedSchemaRef{
					Pointer: joinJSONPointer(pointer, key), Value: ref,
				})
			}
		}
		switch key {
		case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
			children, _ := value.(map[string]any)
			for childName, child := range children {
				if childObject, ok := child.(map[string]any); ok {
					scanContractSchemaObject(childObject, joinJSONPointer(joinJSONPointer(pointer, key), childName), document)
				}
			}
		case "items", "contains", "additionalProperties", "unevaluatedProperties", "unevaluatedItems", "propertyNames", "not", "if", "then", "else", "contentSchema":
			if childObject, ok := value.(map[string]any); ok {
				scanContractSchemaObject(childObject, joinJSONPointer(pointer, key), document)
			}
		case "allOf", "anyOf", "oneOf", "prefixItems":
			children, _ := value.([]any)
			for index, child := range children {
				if childObject, ok := child.(map[string]any); ok {
					scanContractSchemaObject(childObject, joinJSONPointer(joinJSONPointer(pointer, key), strconv.Itoa(index)), document)
				}
			}
		}
	}
}

func joinJSONPointer(base, token string) string {
	escaped := strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
	return base + "/" + escaped
}

func validateContractSchemaID(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return err.Error()
	}
	if !parsed.IsAbs() || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return "ID must be an absolute HTTP(S) URL"
	}
	if parsed.Fragment != "" {
		return "ID must not contain a fragment"
	}
	return ""
}

func resolveContractSchemaRef(source *loadedContractSchema, raw string, pathIndex, idIndex map[string]*loadedContractSchema) (target, fragment, classification, detail string) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", "missing", "invalid reference URI: " + err.Error()
	}
	fragment = parsed.Fragment
	parsed.Fragment = ""
	if parsed.IsAbs() || parsed.Host != "" {
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return "", fragment, "external", "non-HTTP local references are unsupported"
		}
		if local := idIndex[parsed.String()]; local != nil {
			return local.Path, fragment, "ok", ""
		}
		return "", fragment, "external", "absolute network reference is not a registered local schema ID"
	}
	if parsed.Path == "" {
		return source.Path, fragment, "ok", ""
	}
	targetPath := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(source.Path), filepath.FromSlash(parsed.Path))))
	if targetPath == "schemas" || !strings.HasPrefix(targetPath, "schemas/") {
		return "", fragment, "missing", "relative reference escapes the schemas directory"
	}
	if pathIndex[targetPath] == nil {
		return targetPath, fragment, "missing", "referenced local schema does not exist"
	}
	return targetPath, fragment, "ok", ""
}

func schemaFragmentExists(document any, fragment string) bool {
	if fragment == "" {
		return true
	}
	decoded, err := url.PathUnescape(fragment)
	if err != nil {
		return false
	}
	if strings.HasPrefix(decoded, "/") {
		current := document
		for _, token := range strings.Split(strings.TrimPrefix(decoded, "/"), "/") {
			token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
			switch typed := current.(type) {
			case map[string]any:
				value, ok := typed[token]
				if !ok {
					return false
				}
				current = value
			case []any:
				index, err := strconv.Atoi(token)
				if err != nil || index < 0 || index >= len(typed) {
					return false
				}
				current = typed[index]
			default:
				return false
			}
		}
		return true
	}
	return schemaContainsAnchor(document, decoded)
}

func schemaContainsAnchor(value any, anchor string) bool {
	switch typed := value.(type) {
	case map[string]any:
		if typed["$anchor"] == anchor || typed["$dynamicAnchor"] == anchor {
			return true
		}
		for _, child := range typed {
			if schemaContainsAnchor(child, anchor) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if schemaContainsAnchor(child, anchor) {
				return true
			}
		}
	}
	return false
}

func compileContractSchemas(documents []*loadedContractSchema, pathIndex map[string]*loadedContractSchema, manifest *schemaCharacterizationManifest) {
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.AssertVocabs()
	compiler.UseLoader(denyContractSchemaURLLoader{})
	compiler.UseRegexpEngine(compileECMAContractSchemaRegexp)

	aliases := make(map[string]*loadedContractSchema)
	addAlias := func(alias string, document *loadedContractSchema) {
		if alias == "" {
			return
		}
		if previous := aliases[alias]; previous != nil && previous.Path != document.Path {
			manifest.CompileErrors = append(manifest.CompileErrors, schemaCharacterizationIssue{
				Path: document.Path, Value: alias, Detail: "resource alias conflicts with " + previous.Path,
			})
			return
		}
		aliases[alias] = document
	}
	for _, document := range documents {
		addAlias(document.Name, document)
		addAlias(document.ID, document)
	}
	for _, source := range documents {
		if source.ID == "" {
			continue
		}
		base, err := url.Parse(source.ID)
		if err != nil {
			continue
		}
		for _, ref := range source.Refs {
			if ref.Target == "" {
				continue
			}
			parsedRef, err := url.Parse(ref.Value)
			if err != nil || parsedRef.IsAbs() || parsedRef.Path == "" {
				continue
			}
			parsedRef.Fragment = ""
			if target := pathIndex[ref.Target]; target != nil {
				addAlias(base.ResolveReference(parsedRef).String(), target)
			}
		}
	}

	aliasNames := make([]string, 0, len(aliases))
	for alias := range aliases {
		aliasNames = append(aliasNames, alias)
	}
	sort.Strings(aliasNames)
	for _, alias := range aliasNames {
		if err := compiler.AddResource(alias, aliases[alias].CompilerDoc); err != nil {
			manifest.CompileErrors = append(manifest.CompileErrors, schemaCharacterizationIssue{
				Path: aliases[alias].Path, Value: alias, Detail: sanitizeSchemaCompilerDetail("register resource: " + err.Error()),
			})
		}
	}
	for _, document := range documents {
		if _, err := compiler.Compile(document.Name); err != nil {
			manifest.CompileErrors = append(manifest.CompileErrors, schemaCharacterizationIssue{
				Path: document.Path, Value: document.Name, Detail: sanitizeSchemaCompilerDetail("compile: " + err.Error()),
			})
		}
	}
}

func sanitizeSchemaCompilerDetail(detail string) string {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return detail
	}
	variants := []string{
		repoRoot,
		filepath.ToSlash(repoRoot),
		"file:///" + filepath.ToSlash(repoRoot),
	}
	for _, variant := range variants {
		detail = strings.ReplaceAll(detail, variant, "$REPO")
		detail = strings.ReplaceAll(detail, strings.ToLower(variant), "$REPO")
	}
	return detail
}

func supportedContractSchemaKeyword(keyword string) bool {
	switch keyword {
	case "$schema", "$id", "$ref", "$anchor", "$dynamicRef", "$dynamicAnchor", "$vocabulary", "$comment", "$defs",
		"title", "description", "default", "deprecated", "readOnly", "writeOnly", "examples",
		"multipleOf", "maximum", "exclusiveMaximum", "minimum", "exclusiveMinimum",
		"maxLength", "minLength", "pattern", "maxItems", "minItems", "uniqueItems", "maxContains", "minContains",
		"maxProperties", "minProperties", "required", "dependentRequired", "const", "enum", "type",
		"prefixItems", "items", "contains", "additionalProperties", "properties", "patternProperties", "dependentSchemas",
		"propertyNames", "if", "then", "else", "allOf", "anyOf", "oneOf", "not", "unevaluatedItems", "unevaluatedProperties",
		"format", "contentEncoding", "contentMediaType", "contentSchema", "definitions":
		return true
	default:
		return false
	}
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortSchemaIssues(issues []schemaCharacterizationIssue) {
	sort.Slice(issues, func(i, j int) bool {
		left := issues[i].Path + "\x00" + issues[i].Pointer + "\x00" + issues[i].Value + "\x00" + issues[i].Detail
		right := issues[j].Path + "\x00" + issues[j].Pointer + "\x00" + issues[j].Value + "\x00" + issues[j].Detail
		return left < right
	})
}

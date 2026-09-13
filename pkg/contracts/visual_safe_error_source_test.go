package contracts

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func readVisualSafeErrorCodes(readFile func(string) ([]byte, error)) (map[string]struct{}, error) {
	const owner = "types_visual.go"
	source, err := readFile(owner)
	if err != nil {
		return nil, fmt.Errorf("VisualSafeErrorCode owner %s: %w", owner, err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), owner, source, 0)
	if err != nil {
		return nil, fmt.Errorf("VisualSafeErrorCode owner %s: %w", owner, err)
	}
	if parsed.Name.Name != "contracts" {
		return nil, fmt.Errorf("VisualSafeErrorCode owner package=%s, want contracts", parsed.Name.Name)
	}
	codes := make(map[string]struct{})
	for _, declaration := range parsed.Decls {
		constants, ok := declaration.(*ast.GenDecl)
		if !ok || constants.Tok != token.CONST {
			continue
		}
		for _, specification := range constants.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeName, ok := value.Type.(*ast.Ident)
			if !ok || typeName.Name != "VisualSafeErrorCode" {
				continue
			}
			if len(value.Names) != len(value.Values) {
				return nil, fmt.Errorf("VisualSafeErrorCode constants must have explicit values")
			}
			for _, expression := range value.Values {
				literal, ok := expression.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return nil, fmt.Errorf("VisualSafeErrorCode constant must be a string literal: %T", expression)
				}
				decoded, err := strconv.Unquote(literal.Value)
				if err != nil {
					return nil, err
				}
				if _, duplicate := codes[decoded]; duplicate || decoded == "" {
					return nil, fmt.Errorf("VisualSafeErrorCode value %q must be nonempty and unique", decoded)
				}
				codes[decoded] = struct{}{}
			}
		}
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("VisualSafeErrorCode owner has no typed constants")
	}
	return codes, nil
}

const visualSafeErrorMutantEnv = "AUTOSTREAM_VISUAL_SAFE_ERROR_SOURCE_TEST_MUTANT"

// Only negative child tests mutate the bytes read from the actual owner. The
// normal parity test reads types_visual.go unchanged; no enum fixture is copied.
func visualSafeErrorSourceForTest(path string) ([]byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	mutant := os.Getenv(visualSafeErrorMutantEnv)
	if mutant == "" {
		return source, nil
	}
	if mutant == "missing_owner" {
		return nil, os.ErrNotExist
	}
	text := string(source)
	var old, replacement string
	switch mutant {
	case "invalid_source":
		old, replacement = "package contracts", "package"
	case "wrong_package":
		old, replacement = "package contracts", "package other"
	case "wrong_type":
		old, replacement = `VisualSafeErrorCode = "invalid_theme_id"`, `string = "invalid_theme_id"`
	case "non_string":
		old, replacement = `VisualSafeErrorCode = "invalid_theme_id"`, "VisualSafeErrorCode = 1"
	case "missing_code":
		for _, line := range strings.SplitAfter(text, "\n") {
			if strings.Contains(line, `VisualSafeErrorCode = "invalid_theme_id"`) {
				old = line
			}
		}
	case "additional_code":
		return append(source, []byte("\nconst TestAdditionalVisualCode VisualSafeErrorCode = \"test_additional_code\"\n")...), nil
	case "changed_value":
		old, replacement = `"invalid_theme_id"`, `"test_changed_code"`
	case "duplicate_value":
		old, replacement = `"invalid_theme_id"`, `"media_asset_too_large"`
	default:
		return nil, fmt.Errorf("unknown VisualSafeErrorCode source mutant %q", mutant)
	}
	if old == "" || strings.Count(text, old) != 1 {
		return nil, fmt.Errorf("VisualSafeErrorCode mutant %s must replace exactly one actual declaration", mutant)
	}
	return []byte(strings.Replace(text, old, replacement, 1)), nil
}

func TestVisualSafeErrorCodeSourceMutationSensitivity(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, want string }{
		{"missing_owner", "VisualSafeErrorCode owner types_visual.go"},
		{"invalid_source", "VisualSafeErrorCode owner types_visual.go"},
		{"wrong_package", "want contracts"},
		{"wrong_type", "VisualSafeErrorCode count=22"},
		{"non_string", "constant must be a string literal"},
		{"missing_code", "VisualSafeErrorCode count=22"},
		{"additional_code", "VisualSafeErrorCode count=24"},
		{"changed_value", `schema safeError code "invalid_theme_id" has no typed VisualSafeErrorCode constant`},
		{"duplicate_value", "must be nonempty and unique"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(executable, "-test.run=^TestVisualSafeErrorCodeSchemaGoParity$", "-test.count=1")
			command.Env = append(os.Environ(), visualSafeErrorMutantEnv+"="+test.name)
			output, err := command.CombinedOutput()
			if err == nil {
				t.Fatalf("source mutant %s unexpectedly passed", test.name)
			}
			if !strings.Contains(string(output), test.want) {
				t.Fatalf("source mutant %s failed for the wrong reason: %v\n%s", test.name, err, output)
			}
		})
	}
}

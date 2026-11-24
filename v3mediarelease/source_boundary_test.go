package mediarelease

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestMediaReleaseProductionSourceBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		count++
		t.Run(name, func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			checkProductDeclarations(t, f)
		})
	}
	if count == 0 {
		t.Fatal("no production Go files inspected")
	}
}

func checkProductDeclarations(t *testing.T, f *ast.File) {
	t.Helper()
	if f.Name.Name != "mediarelease" {
		t.Error("unexpected package")
	}
	allowed := strings.Fields("bytes crypto/sha256 encoding/hex encoding/json errors fmt io math reflect slices sort strconv strings time unicode unicode/utf8")
	sensitive := strings.Fields("password passwordhash mfasecret recoverycode recoveryhash sessiontoken csrftoken oauthsecret privatekey encryptionkey rawpayload rawaudio rawvideo rawmedia credential credentials notificationbody notificationurl connectionurl accesstoken refreshtoken")
	for _, group := range f.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "//line ") {
				t.Error("production compiler directive")
			}
		}
	}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !slices.Contains(allowed, path) || imp.Name != nil {
			t.Errorf("unexpected production import: %s", imp.Path.Value)
		}
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := strings.ToLower(fn.Name.Name)
		issuer := strings.HasPrefix(name, "new") || strings.HasPrefix(name, "issue") ||
			strings.HasPrefix(name, "mint") || strings.HasPrefix(name, "consume")
		for _, noun := range []string{"authority", "permit", "approval", "recovery", "qualification"} {
			if issuer && strings.Contains(name, noun) {
				t.Errorf("authority constructor: %s", fn.Name.Name)
			}
		}
		if name == "init" || name == "main" || name == "qualifyseal" || name == "qualifyproduction" {
			t.Errorf("executable or qualification entry point: %s", name)
		}
		if name == "authorizesmutation" || name == "productionqualified" {
			if fn.Body == nil || len(fn.Body.List) != 1 {
				t.Fatal("qualification must be constant false")
			}
			ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				t.Fatal("qualification must return false")
			}
			id, ok := ret.Results[0].(*ast.Ident)
			if !ok || id.Name != "false" {
				t.Error("qualification must return literal false")
			}
		}
	}
	ast.Inspect(f, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.StructType:
			for _, field := range n.Fields.List {
				var names []string
				for _, id := range field.Names {
					names = append(names, id.Name)
				}
				if field.Tag != nil {
					tag, err := strconv.Unquote(field.Tag.Value)
					if err != nil {
						t.Fatal(err)
					}
					wire, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
					names = append(names, wire)
				}
				for _, name := range names {
					if slices.Contains(sensitive, strings.ReplaceAll(strings.ToLower(name), "_", "")) {
						t.Errorf("forbidden raw/secret field or tag: %s", name)
					}
				}
			}
		case *ast.SelectorExpr:
			id, ok := n.X.(*ast.Ident)
			if ok && id.Name == "time" && !slices.Contains([]string{"Parse", "Time", "RFC3339Nano", "Duration", "Nanosecond"}, n.Sel.Name) {
				t.Error("production can only parse and compare supplied time")
			}
		case *ast.GoStmt:
			t.Error("background operation in pure contract module")
		}
		return true
	})
}

package commerce_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestBusinessServicesHaveNoAgentOrInfrastructureImports(t *testing.T) {
	for _, dir := range []string{"auth", "catalog", "cart", "order", "promotion", "review", "knowledge", "files"} {
		err := filepath.WalkDir(filepath.Join("internal", dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				value, _ := strconv.Unquote(imp.Path.Value)
				if strings.Contains(value, "/platform/") || strings.Contains(value, "/agent") || strings.HasPrefix(value, "github.com/LYP-leo/xzxg-shop/backend") {
					t.Errorf("%s imports %s", path, value)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

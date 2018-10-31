package langserver

import (
	"context"
	"fmt"
	"github.com/saibing/bingo/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/saibing/bingo/langserver/util"
)

func TestDocumentSymbol(t *testing.T) {
	test := func(t *testing.T, pkgDir string, data map[string][]string) {
		for k, v := range data {
			testDocumentSymbol(t, &documentSymbolTestCase{pkgDir:pkgDir,input:k, output:v})
		}
	}

	t.Run("basic document symbol", func(t *testing.T) {
		test(t, basicPkgDir, map[string][]string{
			"a.go": {"/src/test/pkg/a.go:function:A:1:17"},
			"b.go": {"/src/test/pkg/b.go:function:B:1:17"},
		})
	})

	t.Run("detailed document symbol", func(t *testing.T) {
		test(t, detailedPkgDir, map[string][]string{
			"a.go": {"/src/test/pkg/a.go:field:T.F:1:28", "/src/test/pkg/a.go:class:T:1:17"},
		})
	})

	t.Run("exported defs unexported type", func(t *testing.T) {
		test(t, exportedPkgDir, map[string][]string{
			"a.go": {"/src/test/pkg/a.go:field:t.F:1:28", "/src/test/pkg/a.go:class:t:1:17"},
		})
	})

	t.Run("xtest", func(t *testing.T) {
		test(t, xtestPkgDir, map[string][]string{
			"y_test.go": {"/src/test/pkg/y_test.go:function:Y:1:22"},
			"b_test.go": {"/src/test/pkg/b_test.go:function:Y:1:17"},
		})
	})

	t.Run("subdirectory document symbol", func(t *testing.T) {
		test(t, subdirectoryPkgDir, map[string][]string{
			"a.go":    {"/src/test/pkg/d/a.go:function:A:1:17"},
			"d2/b.go": {"/src/test/pkg/d/d2/b.go:function:B:1:39"},
		})
	})

	t.Run("multiple packages in dir", func(t *testing.T) {
		test(t, multiplePkgDir, map[string][]string{
			"a.go": {"/src/test/pkg/a.go:function:A:1:17"},
		})
	})

	t.Run("go root", func(t *testing.T) {
		test(t, gorootPkgDir, map[string][]string{
			"a.go": {
				"/src/test/pkg/a.go:variable:x:1:51",
			},
		})
	})

	t.Run("go project", func(t *testing.T) {
		test(t, goprojectPkgDir, map[string][]string{
			"a/a.go": {"/src/test/pkg/a/a.go:function:A:1:17"},
			"b/b.go": {},
		})
	})

	t.Run("go symbols", func(t *testing.T) {
		test(t, symbolsPkgDir, map[string][]string{
			"abc.go": {"/src/test/pkg/abc.go:class:XYZ:3:6", "/src/test/pkg/abc.go:method:XYZ.ABC:5:14", "/src/test/pkg/abc.go:variable:A:8:2", "/src/test/pkg/abc.go:constant:B:12:2", "/src/test/pkg/abc.go:class:C:17:2", "/src/test/pkg/abc.go:interface:UVW:20:6", "/src/test/pkg/abc.go:class:T:22:6"},
			"bcd.go": {"/src/test/pkg/bcd.go:class:YZA:3:6", "/src/test/pkg/bcd.go:method:YZA.BCD:5:14"},
			"cde.go": {"/src/test/pkg/cde.go:variable:a:4:2", "/src/test/pkg/cde.go:variable:b:4:5", "/src/test/pkg/cde.go:variable:c:5:2"},
			"xyz.go": {"/src/test/pkg/xyz.go:function:yza:3:6"},
		})
	})

	t.Run("unexpected paths", func(t *testing.T) {
		test(t, unexpectedPkgDir, map[string][]string{
			"a.go": {"/src/t:est/@hello/pkg/a.go:function:A:1:17"},
		})
	})

	t.Run("recv in different file", func(t *testing.T) {
		test(t, differentPkgDir, map[string][]string{
			"abc.go": {"/src/test/pkg/abc.go:class:XYZ:2:6"},
			"bcd.go": {"/src/test/pkg/bcd.go:method:XYZ.ABC:2:14"},
		})
	})
}

type documentSymbolTestCase struct {
	pkgDir string
	input  string
	output []string
}

func testDocumentSymbol(tb testing.TB, c *documentSymbolTestCase) {
	tbRun(tb, fmt.Sprintf("document-symbol-%s", c.input), func(t testing.TB) {
		dir, err := filepath.Abs(c.pkgDir)
		if err != nil {
			log.Fatal("testDocumentSymbol", err)
		}
		doTestDocumentSymbol(t, ctx, conn, util.PathToURI(dir), c.input, c.output)
	})
}

func doTestDocumentSymbol(t testing.TB, ctx context.Context, c *jsonrpc2.Conn, rootURI lsp.DocumentURI, file string, want []string) {
	symbols, err := callSymbols(ctx, c, uriJoin(rootURI, file))
	if err != nil {
		t.Fatal(err)
	}
	for i := range symbols {
		symbols[i] = util.UriToPath(lsp.DocumentURI(symbols[i]))
	}
	if !reflect.DeepEqual(symbols, want) {
		t.Errorf("got %q, want %q", symbols, want)
	}
}

func callSymbols(ctx context.Context, c *jsonrpc2.Conn, uri lsp.DocumentURI) ([]string, error) {
	var symbols []lsp.SymbolInformation
	err := c.Call(ctx, "textDocument/documentSymbol", lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{URI: uri},
	}, &symbols)
	if err != nil {
		return nil, err
	}
	syms := make([]string, len(symbols))
	for i, s := range symbols {
		syms[i] = fmt.Sprintf("%s:%s:%s:%d:%d", s.Location.URI, strings.ToLower(s.Kind.String()), qualifiedName(s), s.Location.Range.Start.Line+1, s.Location.Range.Start.Character+1)
	}
	return syms, nil
}

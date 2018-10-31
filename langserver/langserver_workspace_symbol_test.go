package langserver

import (
	"context"
	"fmt"
	"github.com/saibing/bingo/pkg/lspext"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/saibing/bingo/langserver/util"

	"github.com/saibing/bingo/pkg/lsp"
	"github.com/sourcegraph/jsonrpc2"
)

func TestWorkspaceSymbol(t *testing.T) {
	test := func(t *testing.T, pkgDir string, data map[*lspext.WorkspaceSymbolParams][]string) {
		for k, v := range data {
			testWorkspaceSymbol(t, &workspaceSymbolTestCase{pkgDir: pkgDir, input: k, output: v})
		}
	}

	t.Run("basic workspace symbol", func(t *testing.T) {
		test(t, basicPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}: {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Query: "A"}:           {"/src/test/pkg/a.go:function:A:1:17"},
			{Query: "B"}:           {"/src/test/pkg/b.go:function:B:1:17"},
			{Query: "is:exported"}: {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Query: "dir:/"}:       {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Query: "dir:/ A"}:     {"/src/test/pkg/a.go:function:A:1:17"},
			{Query: "dir:/ B"}:     {"/src/test/pkg/b.go:function:B:1:17"},

			// non-nil SymbolDescriptor + no keys.
			{Symbol: make(lspext.SymbolDescriptor)}: {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},

			// Individual filter fields.
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg"}}: {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"name": "A"}}:           {"/src/test/pkg/a.go:function:A:1:17"},
			{Symbol: lspext.SymbolDescriptor{"name": "B"}}:           {"/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"packageName": "p"}}:    {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"recv": ""}}:            {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"vendor": false}}:       {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},

			// Combined filter fields.
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg"}}:                                                               {"/src/test/pkg/a.go:function:A:1:17", "/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "A"}}:                                                  {"/src/test/pkg/a.go:function:A:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "A", "packageName": "p"}}:                              {"/src/test/pkg/a.go:function:A:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "A", "packageName": "p", "recv": ""}}:                  {"/src/test/pkg/a.go:function:A:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "A", "packageName": "p", "recv": "", "vendor": false}}: {"/src/test/pkg/a.go:function:A:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "B"}}:                                                  {"/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "B", "packageName": "p"}}:                              {"/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "B", "packageName": "p", "recv": ""}}:                  {"/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "B", "packageName": "p", "recv": "", "vendor": false}}: {"/src/test/pkg/b.go:function:B:1:17"},

			// By ID.
			{Symbol: lspext.SymbolDescriptor{"id": "test/pkg/-/B"}}: {"/src/test/pkg/b.go:function:B:1:17"},
			{Symbol: lspext.SymbolDescriptor{"id": "test/pkg/-/A"}}: {"/src/test/pkg/a.go:function:A:1:17"},
		})
	})

	t.Run("detailed workspace symbol", func(t *testing.T) {
		test(t, detailedPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}:            {"/src/test/pkg/a.go:class:T:1:17", "/src/test/pkg/a.go:field:T.F:1:28"},
			{Query: "T"}:           {"/src/test/pkg/a.go:class:T:1:17", "/src/test/pkg/a.go:field:T.F:1:28"},
			{Query: "F"}:           {"/src/test/pkg/a.go:field:T.F:1:28"},
			{Query: "is:exported"}: {"/src/test/pkg/a.go:class:T:1:17", "/src/test/pkg/a.go:field:T.F:1:28"},
		})
	})

	t.Run("exported defs unexported type", func(t *testing.T) {
		test(t, exportedPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: "is:exported"}: {},
		})
	})

	t.Run("subdirectory workspace symbol", func(t *testing.T) {
		test(t, subdirectoryPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}:            {"/src/test/pkg/d/a.go:function:A:1:17", "/src/test/pkg/d/d2/b.go:function:B:1:39"},
			{Query: "is:exported"}: {"/src/test/pkg/d/a.go:function:A:1:17", "/src/test/pkg/d/d2/b.go:function:B:1:39"},
			{Query: "dir:"}:        {"/src/test/pkg/d/a.go:function:A:1:17"},
			{Query: "dir:/"}:       {"/src/test/pkg/d/a.go:function:A:1:17"},
			{Query: "dir:."}:       {"/src/test/pkg/d/a.go:function:A:1:17"},
			{Query: "dir:./"}:      {"/src/test/pkg/d/a.go:function:A:1:17"},
			{Query: "dir:/d2"}:     {"/src/test/pkg/d/d2/b.go:function:B:1:39"},
			{Query: "dir:./d2"}:    {"/src/test/pkg/d/d2/b.go:function:B:1:39"},
			{Query: "dir:d2/"}:     {"/src/test/pkg/d/d2/b.go:function:B:1:39"},
		})
	})

	t.Run("multiple packages in dir", func(t *testing.T) {
		test(t, multiplePkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}:            {"/src/test/pkg/a.go:function:A:1:17"},
			{Query: "is:exported"}: {"/src/test/pkg/a.go:function:A:1:17"},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "A", "packageName": "p", "recv": "", "vendor": false}}: {"/src/test/pkg/a.go:function:A:1:17"},
		})
	})

	t.Run("go root", func(t *testing.T) {
		test(t, gorootPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}: {
				"/src/test/pkg/a.go:variable:x:1:51",
			},
			{Query: "is:exported"}: {},
			{Symbol: lspext.SymbolDescriptor{"package": "test/pkg", "name": "x", "packageName": "p", "recv": "", "vendor": false}}: {"/src/test/pkg/a.go:variable:x:1:51"},
		})
	})

	t.Run("go project", func(t *testing.T) {
		test(t, goprojectPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}:            {"/src/test/pkg/a/a.go:function:A:1:17"},
			{Query: "is:exported"}: {"/src/test/pkg/a/a.go:function:A:1:17"},
		})
	})

	t.Run("go symbols", func(t *testing.T) {
		test(t, symbolsPkgDir, map[*lspext.WorkspaceSymbolParams][]string{
			{Query: ""}:            {"/src/test/pkg/abc.go:variable:A:8:2", "/src/test/pkg/abc.go:constant:B:12:2", "/src/test/pkg/abc.go:class:C:17:2", "/src/test/pkg/abc.go:class:T:22:6", "/src/test/pkg/abc.go:interface:UVW:20:6", "/src/test/pkg/abc.go:class:XYZ:3:6", "/src/test/pkg/bcd.go:class:YZA:3:6", "/src/test/pkg/cde.go:variable:a:4:2", "/src/test/pkg/cde.go:variable:b:4:5", "/src/test/pkg/cde.go:variable:c:5:2", "/src/test/pkg/xyz.go:function:yza:3:6", "/src/test/pkg/abc.go:method:XYZ.ABC:5:14", "/src/test/pkg/bcd.go:method:YZA.BCD:5:14"},
			{Query: "xyz"}:         {"/src/test/pkg/abc.go:class:XYZ:3:6", "/src/test/pkg/abc.go:method:XYZ.ABC:5:14", "/src/test/pkg/xyz.go:function:yza:3:6"},
			{Query: "yza"}:         {"/src/test/pkg/bcd.go:class:YZA:3:6", "/src/test/pkg/xyz.go:function:yza:3:6", "/src/test/pkg/bcd.go:method:YZA.BCD:5:14"},
			{Query: "abc"}:         {"/src/test/pkg/abc.go:method:XYZ.ABC:5:14", "/src/test/pkg/abc.go:variable:A:8:2", "/src/test/pkg/abc.go:constant:B:12:2", "/src/test/pkg/abc.go:class:C:17:2", "/src/test/pkg/abc.go:class:T:22:6", "/src/test/pkg/abc.go:interface:UVW:20:6", "/src/test/pkg/abc.go:class:XYZ:3:6"},
			{Query: "bcd"}:         {"/src/test/pkg/bcd.go:method:YZA.BCD:5:14", "/src/test/pkg/bcd.go:class:YZA:3:6"},
			{Query: "cde"}:         {"/src/test/pkg/cde.go:variable:a:4:2", "/src/test/pkg/cde.go:variable:b:4:5", "/src/test/pkg/cde.go:variable:c:5:2"},
			{Query: "is:exported"}: {"/src/test/pkg/abc.go:variable:A:8:2", "/src/test/pkg/abc.go:constant:B:12:2", "/src/test/pkg/abc.go:class:C:17:2", "/src/test/pkg/abc.go:class:T:22:6", "/src/test/pkg/abc.go:interface:UVW:20:6", "/src/test/pkg/abc.go:class:XYZ:3:6", "/src/test/pkg/bcd.go:class:YZA:3:6", "/src/test/pkg/abc.go:method:XYZ.ABC:5:14", "/src/test/pkg/bcd.go:method:YZA.BCD:5:14"},
		})
	})
}

type workspaceSymbolTestCase struct {
	pkgDir string
	input  *lspext.WorkspaceSymbolParams
	output []string
}

func testWorkspaceSymbol(tb testing.TB, c *workspaceSymbolTestCase) {
	tbRun(tb, fmt.Sprintf("workspace-symbol-%s", c.input.Query), func(t testing.TB) {
		dir, err := filepath.Abs(c.pkgDir)
		if err != nil {
			log.Fatal("testWorkspaceSymbol", err)
		}
		doWorkspaceSymbolsTest(t, ctx, conn, util.PathToURI(dir), *c.input, c.output)
	})
}

func doWorkspaceSymbolsTest(t testing.TB, ctx context.Context, c *jsonrpc2.Conn, rootURI lsp.DocumentURI, params lspext.WorkspaceSymbolParams, want []string) {
	symbols, err := callWorkspaceSymbols(ctx, c, params)
	if err != nil {
		t.Fatal(err)
	}
	for i := range symbols {
		symbols[i] = util.UriToPath(lsp.DocumentURI(symbols[i]))
	}
	if !reflect.DeepEqual(symbols, want) {
		t.Errorf("got %#v, want %q", symbols, want)
	}
}

func callWorkspaceSymbols(ctx context.Context, c *jsonrpc2.Conn, params lspext.WorkspaceSymbolParams) ([]string, error) {
	var symbols []lsp.SymbolInformation
	err := c.Call(ctx, "workspace/symbol", params, &symbols)
	if err != nil {
		return nil, err
	}
	syms := make([]string, len(symbols))
	for i, s := range symbols {
		syms[i] = fmt.Sprintf("%s:%s:%s:%d:%d", s.Location.URI, strings.ToLower(s.Kind.String()), qualifiedName(s), s.Location.Range.Start.Line+1, s.Location.Range.Start.Character+1)
	}
	return syms, nil
}
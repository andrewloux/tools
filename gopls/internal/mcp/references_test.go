// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/internal/cache"
	"golang.org/x/tools/gopls/internal/protocol"
)

func TestFormatReferences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	if err := os.WriteFile(path, []byte("package a\n\n\tFoo(); Foo()\n"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.go")
	location := func(path string, line, character uint32) protocol.Location {
		return protocol.Location{
			URI: protocol.URIFromPath(path),
			Range: protocol.Range{Start: protocol.Position{
				Line: line, Character: character,
			}},
		}
	}
	session := cache.NewSession(t.Context(), cache.New(nil))
	t.Cleanup(func() { session.Shutdown(context.Background()) })

	// Retain distinct references on one line and locations whose source text
	// is unavailable, whether the file is missing or the line is out of range.
	refs := []protocol.Location{
		location(path, 2, 1),
		location(path, 2, 8),
		location(missing, 4, 2),
		location(path, 10, 0),
	}
	result, err := formatReferences(t.Context(), session, refs)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("References: 4\n%s:3:2\tFoo(); Foo()\n%s:3:9\tFoo(); Foo()\n%s:5:3\n%s:11:1\n",
		filepath.ToSlash(path), filepath.ToSlash(path), filepath.ToSlash(missing), filepath.ToSlash(path))
	if got := result.Content[0].(*mcp.TextContent).Text; got != want {
		t.Errorf("reference output:\ngot  %q\nwant %q", got, want)
	}
	if _, err := formatReferences(t.Context(), session, nil); err == nil {
		t.Error("expected an error for no references")
	}
}

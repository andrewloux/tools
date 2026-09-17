// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/tools/gopls/internal/file"
	"golang.org/x/tools/gopls/internal/golang"
	"golang.org/x/tools/gopls/internal/protocol"
)

type findReferencesParams struct {
	Location protocol.Location `json:"location"`
}

func (h *handler) referencesHandler(ctx context.Context, req *mcp.CallToolRequest, params findReferencesParams) (*mcp.CallToolResult, any, error) {
	countGoReferencesMCP.Inc()
	fh, snapshot, release, err := h.session.FileOf(ctx, params.Location.URI)
	if err != nil {
		return nil, nil, err
	}
	defer release()
	pos := params.Location.Range.Start
	refs, err := golang.References(ctx, snapshot, fh, protocol.Range{Start: pos, End: pos}, true)
	if err != nil {
		return nil, nil, err
	}
	formatted, err := formatReferences(ctx, snapshot, refs)
	return formatted, nil, err
}

func formatReferences(ctx context.Context, fs file.Source, refs []protocol.Location) (*mcp.CallToolResult, error) {
	if len(refs) == 0 {
		return nil, fmt.Errorf("no references found")
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "References: %d", len(refs))
	for _, r := range refs {
		fmt.Fprintf(&builder, "\n%s:%d:%d", filepath.ToSlash(r.URI.Path()), r.Range.Start.Line+1, r.Range.Start.Character+1)
		// Keep the location even when its source text is unavailable.
		refFh, err := fs.ReadFile(ctx, r.URI)
		if err != nil {
			continue
		}
		content, err := refFh.Content()
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		var lineContent string
		if int(r.Range.Start.Line) < len(lines) {
			lineContent = strings.TrimLeftFunc(lines[r.Range.Start.Line], unicode.IsSpace)
		} else {
			continue
		}
		fmt.Fprintf(&builder, "\t%s", lineContent)
	}
	builder.WriteByte('\n')
	return textResult(builder.String()), nil
}

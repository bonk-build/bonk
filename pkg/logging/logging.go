// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package logging

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/pterm/pterm"

	slogmulti "github.com/samber/slog-multi"

	"go.bonk.build/pkg/executor/tree"
)

// build a sick terminal ux

var sep = ": "

type handler struct {
	mu          sync.Mutex
	area        pterm.AreaPrinter
	treePrinter pterm.TreePrinter
	treeNode    pterm.TreeNode

	allElse io.Writer
}

func NewHandler() slog.Handler {
	multi := pterm.DefaultMultiPrinter

	h := &handler{
		area:        pterm.DefaultArea,
		treePrinter: *pterm.DefaultTree.WithWriter(multi.NewWriter()),
		treeNode:    pterm.TreeNode{},

		allElse: multi.NewWriter(),
	}

	multi.Start()

	return slogmulti.NewHandleInlineHandler(
		func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
			h.mu.Lock()
			defer h.mu.Unlock()

			var tsk string
			record.Attrs(func(a slog.Attr) bool {
				if a.Key == "task" {
					tsk = a.Value.String()

					return false
				}

				return true
			})

			if tsk != "" {
				cur := &h.treeNode
				for {
					before, after, hasSep := strings.Cut(tsk, tree.ExecPathSep)

					idx := slices.IndexFunc(cur.Children, func(n pterm.TreeNode) bool {
						return n.Text == before || strings.HasPrefix(n.Text, before+sep)
					})

					if idx == -1 {
						cur.Children = append(cur.Children, pterm.TreeNode{
							Text: pterm.Gray(before),
						})
						cur = &cur.Children[len(cur.Children)-1]
					}

					if hasSep {
						tsk = after
					} else {
						break
					}
				}
				before, _, _ := strings.Cut(cur.Text, sep)
				cur.Text = before + sep + record.Message

				h.treePrinter.WithRoot(h.treeNode).Render()
			} else {
				h.allElse.Write([]byte(record.Message + "\n"))
			}

			return nil
		},
	)
}

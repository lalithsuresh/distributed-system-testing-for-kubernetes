// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package tracker

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"kubetorch/analysispasses/collector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "tracker",
	Doc:      "track the handlers from the collector",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer, collector.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	handlers := pass.ResultOf[collector.Analyzer].([]string)
	if len(handlers) == 0 {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fd := n.(*ast.FuncDecl)
		for _, handler := range handlers {
			if fd.Name.Name == handler {
				pass.Reportf(n.Pos(), "fun: %s", fd.Name.Name)
			}
		}
	})

	return nil, nil
}

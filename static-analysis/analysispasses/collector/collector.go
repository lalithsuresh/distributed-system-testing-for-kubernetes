// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package collector

import (
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"reflect"
	"strings"
)

var Analyzer = &analysis.Analyzer{
	Name:       "collector",
	Doc:        "find all AddEventHandler for informers in Kubernetes",
	Run:        run,
	ResultType: reflect.TypeOf([]string{}),
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {

	handlers := make([]string, 0)
	if strings.HasSuffix(pass.Pkg.Path(), ".test") {
		return handlers, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	extractHandlers := func(e ast.Expr) {
		v := e.(*ast.KeyValueExpr).Value
		switch v.(type) {
		case *ast.SelectorExpr:
			se := v.(*ast.SelectorExpr)
			pass.Reportf(e.Pos(), "handler: %v.%v", se.X, se.Sel)
			handlers = append(handlers, se.Sel.Name)
		case *ast.FuncLit:
			pass.Reportf(e.Pos(), "doesn't support FuncLit now")
		default:
			pass.Reportf(e.Pos(), "not supported")
		}
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ce := n.(*ast.CallExpr)
		se, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if se.Sel.Name == "AddEventHandler" {
			pass.Reportf(ce.Pos(), "call: %v", ce.Fun)
			cpl, ok := ce.Args[0].(*ast.CompositeLit)
			if !ok {
				return
			}
			switch cpl.Type.(*ast.SelectorExpr).Sel.Name {
			case "FilteringResourceEventHandler":
				for _, elt := range cpl.Elts[1].(*ast.KeyValueExpr).Value.(*ast.CompositeLit).Elts {
					extractHandlers(elt)
				}
			case "ResourceEventHandlerFuncs":
				for _, elt := range cpl.Elts {
					extractHandlers(elt)
				}
			default:
				pass.Reportf(ce.Pos(), "not support yet")
				return
			}
		}
	})
	fmt.Println(len(handlers))
	return handlers, nil
}

// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package collector

import (
	"fmt"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"log"
	"strings"
)

const (
	FREH = "FilteringResourceEventHandler"
	REH  = "ResourceEventHandlerFuncs"
)

type Collector struct {
	pattern    string
	prog       *ssa.Program
	handlerMap map[*ssa.Call]map[string]*ssa.Function
}

func (c *Collector) extractFREHandlers(root *ssa.Alloc) map[string]*ssa.Function {
	fa := (*root.Referrers())[1].(*ssa.FieldAddr)
	st := (*fa.Referrers())[0].(*ssa.Store)
	allocHandler := st.Val.(*ssa.MakeInterface).X.(*ssa.UnOp).X.(*ssa.Alloc)
	return c.extractREHandlers(allocHandler)
}

func (c *Collector) extractREHandlers(root *ssa.Alloc) map[string]*ssa.Function {
	m := map[string]*ssa.Function{}

	handlerType := func(s string) string {
		if strings.Contains(s, "AddFunc") {
			return "Add"
		} else if strings.Contains(s, "UpdateFunc") {
			return "Update"
		} else if strings.Contains(s, "DeleteFunc") {
			return "Delete"
		} else {
			return "Unknown"
		}
	}

	for _, instr := range *root.Referrers() {
		fa, ok := instr.(*ssa.FieldAddr)
		if !ok {
			continue
		}
		st := (*fa.Referrers())[0].(*ssa.Store)
		m[handlerType(fa.String())] = st.Val.(*ssa.MakeClosure).Fn.(*ssa.Function)
	}
	//fmt.Println(m)
	return m
}

func (c *Collector) extractHandlers(prog *ssa.Program, pattern string) map[*ssa.Call]map[string]*ssa.Function {
	m := map[*ssa.Call]map[string]*ssa.Function{}
	for _, pkg := range prog.AllPackages() {
		if pkg.Pkg.Path() == pattern {
			fun := pkg.Func("addAllEventHandlers") // Hardcoded here. Relax it later.
			//fun.WriteTo(os.Stdout)
			for _, block := range fun.Blocks {
				for _, instr := range block.Instrs {
					call, ok := instr.(*ssa.Call)
					if !ok {
						continue
					}
					if call.Common().IsInvoke() && call.Common().Method.Name() == "AddEventHandler" { // Hardcoded here. Relax it later.
						//fmt.Println(call)
						switch call.Common().Args[0].(type) {
						case *ssa.MakeInterface:
							mi := call.Common().Args[0].(*ssa.MakeInterface)
							allocHandler := mi.X.(*ssa.UnOp).X.(*ssa.Alloc)
							allocHandler.Pos()
							handlerType := allocHandler.Type().String()
							if strings.HasSuffix(handlerType, FREH) {
								//fmt.Println("handle FilteringResourceEventHandler")
								m[call] = c.extractFREHandlers(allocHandler)
							} else if strings.HasSuffix(handlerType, REH) {
								//fmt.Println("handle ResourceEventHandlerFunc")
								m[call] = c.extractREHandlers(allocHandler)
							}
						default:
							fmt.Println("doesn't support")
						}
					}
				}
			}
		}
	}
	//fmt.Println(m)
	return m
}

func (c *Collector) CollectEntryPoints() {
	cfg := packages.Config{Mode: packages.LoadAllSyntax}
	initial, err := packages.Load(&cfg, c.pattern)
	if err != nil {
		log.Fatal(err)
	}
	prog, _ := ssautil.AllPackages(initial, 0)
	prog.Build()
	c.prog = prog
	c.handlerMap = c.extractHandlers(prog, c.pattern)
}

func (c *Collector) GetPattern() string {
	return c.pattern
}

func (c *Collector) GetProg() *ssa.Program {
	return c.prog
}

func (c *Collector) GetHandlerMap() map[*ssa.Call]map[string]*ssa.Function {
	return c.handlerMap
}

func NewCollector(pattern string) *Collector {
	c := &Collector{
		pattern:    pattern,
		handlerMap: map[*ssa.Call]map[string]*ssa.Function{},
	}
	return c
}

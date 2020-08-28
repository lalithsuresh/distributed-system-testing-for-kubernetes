// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package tracker

import (
	"fmt"
	"github.com/golang-collections/go-datastructures/queue"
	"go/types"
	"golang.org/x/tools/go/ssa"
	"kubetorch/ssapasses/collector"
	"strings"
)

type Tracker struct {
	pattern    string
	prog       *ssa.Program
	handlerMap map[*ssa.Call]map[string]*ssa.Function
	methodMap  map[string][]*ssa.Function
	endpoints  map[string]struct{}
}

const separator = "========================================================================="

func (t *Tracker) isWritten(fa *ssa.FieldAddr) bool {
	if len(*fa.Referrers()) != 1 {
		panic("FieldAddr's referrer should be only 1")
	}
	instr := (*fa.Referrers())[0]
	switch instr.(type) {
	case *ssa.Store:
		st := instr.(*ssa.Store)
		if st.Addr == fa {
			return true
		}
	case *ssa.UnOp:
		uo := instr.(*ssa.UnOp)
		uoR := (*uo.Referrers())[0]
		if invoke, ok := uoR.(*ssa.Call); ok && invoke.Common().IsInvoke() && invoke.Common().Value == uo {
			return true
		}
	default:
		return false
	}
	return false
}

func (t *Tracker) trackSingleFunction(function *ssa.Function, funQ *queue.Queue, writtenMembers map[*ssa.FieldAddr]struct{}) {
	if function.Signature.Recv() != nil {
		//fmt.Println("type: ", function.Params[0].Type().String())
		for _, instr := range *(function.Params[0].Referrers()) {
			switch instr.(type) {
			case *ssa.FieldAddr:
				//fmt.Printf("FA: %v\n", instr)
				fa := instr.(*ssa.FieldAddr)
				if t.isWritten(fa) {
					writtenMembers[fa] = struct{}{}
				}
			default:
			}
		}
	}

	for _, block := range function.Blocks {
		for _, instr := range block.Instrs {
			switch instr.(type) {
			case *ssa.Call:
				call := instr.(*ssa.Call)
				if !call.Common().IsInvoke() {
					// TODO: relax it later.
					if strings.Contains(call.Common().String(),
						"k8s.io/kubernetes/pkg/scheduler.Scheduler") {
						funQ.Put(call.Common().StaticCallee())
					}
				}
			default:
			}
		}
	}
}

func (t *Tracker) findReadPoints(function *ssa.Function, writtenMembers map[*ssa.FieldAddr]struct{}, readMap map[*ssa.Function][]ssa.Value) {
	//fmt.Println("finding read points in method: ", function.Name())
	for _, block := range function.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if ok && call.Common().IsInvoke() {
				// TODO: relax it later. So far let's only care about the method "invoke Schedule"
				if call.Common().Method.Name() == "Schedule" {
					//fmt.Println(call)
					for member := range writtenMembers {
						// TODO: relax it later. It is very challenging to determine whether one member is read here
						if member.X.Type().String() == "*k8s.io/kubernetes/pkg/scheduler.Scheduler" &&
							member.Field == 0 {
							readMap[function] = append(readMap[function], call)
						}
					}
				}
			}
		}
	}
}

func (t *Tracker) trackReadPointWithinMethod(fun *ssa.Function, taintedVarsFromOuter map[ssa.Value]struct{}) []ssa.Instruction {

	endpoints := []ssa.Instruction{}
	taintedVars := taintedVarsFromOuter
	//fmt.Println( "init tainted for ", fun.String(), " is ", taintedVars)

	referrerQ := queue.New(100)

	for key := range taintedVars {
		for _, ref := range *(key.Referrers()) {
			referrerQ.Put(ref)
		}
	}

	for !referrerQ.Empty() {
		refs, _ := referrerQ.Get(1)
		ref := refs[0].(ssa.Instruction)
		//fmt.Println(ref)
		switch ref.(type) {
		case *ssa.Alloc:
			al := ref.(*ssa.Alloc)
			if _, found := t.endpoints[al.String()]; found {
				//fmt.Println("Reach endpoint!!!", al)
				endpoints = append(endpoints, al)
			}
			taintedVars[al] = struct{}{}
			for _, rref := range *(al.Referrers()) {
				referrerQ.Put(rref)
			}
		case *ssa.Extract:
			ex := ref.(*ssa.Extract)
			taintedVars[ex] = struct{}{}
			for _, rref := range *(ex.Referrers()) {
				referrerQ.Put(rref)
			}
		case *ssa.Store:
			st := ref.(*ssa.Store)
			stAddr := st.Addr
			_, foundVal := taintedVars[st.Val]
			_, foundAddr := taintedVars[st.Addr]

			// Store is very ticky here. Think about the example:
			// ...
			// t1 = new ...
			// t2 = &t1.foo [#2]
			// t3 = &t2.bar [#3]
			// *t3 = tainted
			// ...
			// if "tainted" is a tainted variable, then t1 is also tainted here
			// We need to do backtrack starting from st.Addr

			if foundVal && !foundAddr {
				taintedVars[st.Addr] = struct{}{}
				back := stAddr
				fa, fok := back.(*ssa.FieldAddr)
				for fok {
					back = fa.X
					fa, fok = back.(*ssa.FieldAddr)
				}
				if nw, ok := back.(*ssa.Alloc); ok {
					referrerQ.Put(nw)
				}
			}
		case *ssa.FieldAddr:
			fa := ref.(*ssa.FieldAddr)
			taintedVars[fa] = struct{}{}
			for _, rref := range *(fa.Referrers()) {
				referrerQ.Put(rref)
			}
		case *ssa.UnOp:
			uo := ref.(*ssa.UnOp)
			taintedVars[uo] = struct{}{}
			for _, rref := range *(uo.Referrers()) {
				referrerQ.Put(rref)
			}
		case *ssa.Call:
			// conservative here: we don't track the referrers of Call for now
			call := ref.(*ssa.Call)
			innerTaintedVars := make(map[ssa.Value]struct{})
			if call.Common().IsInvoke() {
				// not support yet
			} else {
				// TODO: relax it later. So far hardcode "bind" as the end point
				if callee, ok := call.Common().Value.(*ssa.Function); ok {
					for i, ap := range call.Common().Args {
						if _, found := taintedVars[ap]; found {
							innerTaintedVars[callee.Params[i]] = struct{}{}
						}
					}
					//fmt.Println("tainted var for callee: ", innerTaintedVars)
					innerEndPoints := t.trackReadPointWithinMethod(callee, innerTaintedVars)
					endpoints = append(endpoints, innerEndPoints...)
				}
			}
		case *ssa.MakeClosure:
			// conservative here: we don't track the referrers of MakeClosure for now
			mc := ref.(*ssa.MakeClosure)
			innerTaintedVars := make(map[ssa.Value]struct{})
			innerFun := mc.Fn.(*ssa.Function)
			for i, binding := range mc.Bindings {
				if _, found := taintedVars[binding]; found {
					innerTaintedVars[innerFun.FreeVars[i]] = struct{}{}
				}
			}
			//fmt.Println("tainted var for inner func: ", innerTaintedVars)
			innerEndPoints := t.trackReadPointWithinMethod(innerFun, innerTaintedVars)
			endpoints = append(endpoints, innerEndPoints...)
		default:
		}
	}
	//fmt.Println( "final tainted for ", fun.String(), " is ", taintedVars)
	return endpoints
}

func (t *Tracker) trackSingleEntryPoint(function *ssa.Function) {

	//fmt.Println(separator)
	// For each handler, we find all the struct members written by the handler (recursively)
	funQ := queue.New(100)
	writtenMembers := make(map[*ssa.FieldAddr]struct{})
	funQ.Put(function)
	for !funQ.Empty() {
		funs, _ := funQ.Get(1)
		f := funs[0].(*ssa.Function)
		t.trackSingleFunction(f, funQ, writtenMembers)
	}
	//fmt.Println("WRITTENMEMBERS for", function.Name(), ":")
	//fmt.Println(writtenMembers)

	//fmt.Println(separator)
	// For each written member, we visit all the methods (from the same struct as the handler) and find the read points
	readMap := make(map[*ssa.Function][]ssa.Value)
	// TODO: relax it later. Answer the question that why we care about Scheduler's methods
	for _, method := range t.methodMap["k8s.io/kubernetes/pkg/scheduler.Scheduler"] {
		// TODO: relax it later. So far let's only care about method "scheduleOne"
		if method.Name() != "scheduleOne" {
			continue
		}
		t.findReadPoints(method, writtenMembers, readMap)
	}
	//fmt.Println("READMAP for writtenmembers from", function.Name(), ":")
	//fmt.Println(readMap)

	fmt.Println(separator)
	endpoints := []ssa.Instruction{}
	for f := range readMap {
		taintedVars := make(map[ssa.Value]struct{})
		for _, readPoint := range readMap[f] {
			taintedVars[readPoint] = struct{}{}
		}
		subEndPoints := t.trackReadPointWithinMethod(f, taintedVars)
		endpoints = append(endpoints, subEndPoints...)
	}
	//fmt.Println("ENDPOINTS reached from", function.Name(), ":")
	fmt.Println("HINT: node resources and pod resources could be changed as the side effects of", function.Name(), "by:")
	fmt.Println(endpoints)
}

func (t *Tracker) generateMethodMap() {
	for _, pkg := range t.prog.AllPackages() {
		if pkg.Pkg.Path() == t.pattern {
			for _, member := range pkg.Members {
				if tm, ok := member.(*ssa.Type); ok {
					//fmt.Println(tm.Type().String())
					methodList := []*ssa.Function{}
					ms1 := t.prog.MethodSets.MethodSet(tm.Type())
					ms2 := t.prog.MethodSets.MethodSet(types.NewPointer(tm.Type()))
					for i := 0; i < ms1.Len(); i = i + 1 {
						methodList = append(methodList, t.prog.MethodValue(ms1.At(i)))
					}
					for i := 0; i < ms2.Len(); i = i + 1 {
						t.prog.MethodValue(ms2.At(i))
						methodList = append(methodList, t.prog.MethodValue(ms2.At(i)))
					}
					t.methodMap[tm.Type().String()] = methodList
				}
			}
		}
	}
}

func (t *Tracker) TrackEntryPoints(targetHandler string) {
	for _, singleMap := range t.handlerMap {
		for _, handler := range singleMap {
			if handler.Name() == targetHandler+"$bound" {
				t.trackSingleEntryPoint(handler)
			}
		}
	}
}

func NewTracker(c *collector.Collector) *Tracker {
	t := &Tracker{
		pattern:    c.GetPattern(),
		prog:       c.GetProg(),
		handlerMap: c.GetHandlerMap(),
		methodMap:  map[string][]*ssa.Function{},
		endpoints:  map[string]struct{}{},
	}
	t.generateMethodMap()
	t.endpoints["new k8s.io/kubernetes/vendor/k8s.io/api/core/v1.Binding (complit)"] = struct{}{}

	return t
}

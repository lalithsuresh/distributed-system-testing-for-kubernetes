// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package collector

import (
	"fmt"
	"golang.org/x/tools/go/analysis/analysistest"
	"os"
	"testing"
)

func TestCollector(t *testing.T) {
	fmt.Println(os.Getenv("GOPATH"))
	analysistest.Run(t, analysistest.TestData(), Analyzer)
}

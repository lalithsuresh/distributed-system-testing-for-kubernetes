// Copyright 2020 VMware, Inc.
//
// SPDX-License-Identifier: BSD-2

package collector

import (
	"testing"
)

func TestCollector(t *testing.T) {
	c := NewCollector("kubetorch/ssapasses/collector/testdata")
	c.CollectEntryPoints()
	m := c.GetHandlerMap()
	if len(m) != 2 {
		t.Errorf("entry point map len should be 2, but %d actually", len(m))
	}

	for _, subm := range m {
		if len(subm) != 3 {
			t.Errorf("entry point sub map len should be 3, but %d actually", len(subm))
		}
	}
}

package main

import (
	"fmt"
	"testing"

	"github.com/sealuzh/goabs/data"
)

func TestRmit(t *testing.T) {
	funCount := 10
	funcs := make([]data.Function, funCount)
	for i := 0; i < funCount; i++ {
		funcs[i] = fun(i)
	}

	iFuns := rmitFuncs(funcs)

	for i, f := range iFuns {
		fmt.Printf("%d - %s\n", i, f.Name)
	}
}

func fun(n int) data.Function {
	return data.Function{
		Name: fmt.Sprintf("Func%d", n),
	}
}

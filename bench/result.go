package bench

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	timeUnit   = "ns/op"
	bytesUnit  = "B/op"
	allocsUnit = "allocs/op"
)

type resultNotParsable error

type result struct {
	Invocations int     // invocations per benchmark execution
	Runtime     float32 // unit: ns/op
	Memory      int     // unit: B/op
	Allocations int     // unit: allocs/op
}

type resultParser interface {
	parse(string) ([]result, error)
}

type rtResultParser struct{}

func (p rtResultParser) parse(s string) ([]result, error) {
	return parse(s, false)
}

type memResultParser struct{}

func (p memResultParser) parse(s string) ([]result, error) {
	return parse(s, true)
}

func parse(s string, mem bool) ([]result, error) {
	resArr := strings.Fields(s)

	ret := []result{}
	emptyResult := []result{}
	curr := result{}

	for i, f := range resArr {
		switch f {
		case timeUnit:
			// parse timeunit
			rt, err := strconv.ParseFloat(resArr[i-1], 32)
			if err != nil {
				return emptyResult, resultNotParsable(fmt.Errorf("Could not parse benchmark result. Error: %v", err))
			}
			curr.Runtime = float32(rt)

			// parse invocation cound
			ivs, err := strconv.ParseInt(resArr[i-2], 10, 64)
			if err != nil {
				return emptyResult, resultNotParsable(fmt.Errorf("Could not parse invocation count. Error: %v", err))
			}
			curr.Invocations = int(ivs)

			// add parsed result to return slice (no further results for that execution)
			if !mem {
				ret = append(ret, curr)
			}
		case bytesUnit:
			b, err := strconv.Atoi(resArr[i-1])
			if err != nil {
				return emptyResult, resultNotParsable(fmt.Errorf("Could not parse benchmark result. Error: %v", err))
			}
			curr.Memory = b
		case allocsUnit:
			a, err := strconv.Atoi(resArr[i-1])
			if err != nil {
				return emptyResult, resultNotParsable(fmt.Errorf("Could not parse benchmark result. Error: %v", err))
			}
			curr.Allocations = a
			// last result for this execution -> add to return slice
			if mem {
				ret = append(ret, curr)
			}
		}
	}
	return ret, nil
}

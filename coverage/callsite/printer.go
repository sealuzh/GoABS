package callsite

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	OutTypeLine = "line"
	OutTypeJson = "json"
)

var _ Printer = jsonPrinter{}
var _ Printer = linePrinter{}

type Printer interface {
	Print()
}

type basePrinter struct {
	out io.Writer
	css List
}

func NewJsonPrinter(out io.Writer, css List) *jsonPrinter {
	e := json.NewEncoder(out)
	return &jsonPrinter{
		basePrinter: basePrinter{
			out: out,
			css: css,
		},
		encoder: e,
	}
}

type jsonPrinter struct {
	basePrinter
	encoder *json.Encoder
}

func (p jsonPrinter) Print() {
	err := p.encoder.Encode(p.css)
	if err != nil {
		fmt.Fprintf(p.out, "Could not write JSON to output:\n %v\n\n", err)
	}
}

func NewLinePrinter(out io.Writer, css List) *linePrinter {
	return &linePrinter{
		basePrinter: basePrinter{
			out: out,
			css: css,
		},
	}
}

type linePrinter struct {
	basePrinter
}

func (p linePrinter) Print() {
	for _, cs := range p.css {
		fmt.Fprintf(p.out, "%s > %s\n", cs.Caller, cs.Callee)
	}
}

package callsite

import "github.com/sealuzh/goabs/data"

type List []Element

type Element struct {
	Caller data.Function `json:"caller"`
	Callee data.Function `json:"callee"`
}

type Finder interface {
	Parse() error
	All() (List, error)
	Package(path string) (List, error)
}

const (
	NotParsedError   Error = "not parsed"
	PkgNotFoundError Error = "pkg not found in callsites"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

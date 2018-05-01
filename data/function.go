package data

import (
	"fmt"
)

// Function represents a Go function.
// It does not contain function parameters nor return types, as they are not part of the function signature.
// The method receiver is part of the signature (if available).
type Function struct {
	Pkg       string `json:"pkg"`
	File      string `json:"file"`
	Name      string `json:"name"`
	Receiver  string `json:"recv"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func (f Function) String() string {
	funcName := f.Name
	if f.Receiver != "" {
		funcName = fmt.Sprintf("(%s).%s", f.Receiver, f.Name)
	}
	var s string
	if f.File == "" {
		s = fmt.Sprintf("%s.%s", f.Pkg, funcName)
	} else {
		s = fmt.Sprintf("%s.{%s}.%s", f.Pkg, f.File, funcName)
	}
	return s
}

func (f Function) StringWithLines() string {
	s := f.String()
	if f.StartLine != -1 && f.EndLine != -1 {
		return fmt.Sprintf("%s:%d-%d", s, f.StartLine, f.EndLine)
	}
	return s
}

type File []Function

type FileMap map[string]File

type PackageMap map[string]FileMap

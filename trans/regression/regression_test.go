package regression

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"bitbucket.org/sealuzh/goptc/data"
)

const (
	src = `
	package regression
	
	import "fmt"

	func test() string {
		fmt.Println("test func")
		return ""
	}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"
		
	func test() string {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
		return ""
	}
	`
)

func fun(pkg, file string) data.Function {
	return data.Function{
		Name: "test",
		Pkg:  pkg,
		File: file,
	}
}

func TestRelRegIntroFunc(t *testing.T) {
	srcOut := srcExpected

	fset := token.NewFileSet()
	e, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		t.Errorf("Could not parse file: %v", err)
		return
	}

	visitor := &relRegVisitor{
		fun:       fun("", ""),
		violation: 1.0,
	}

	ast.Walk(visitor, e)

	var buf bytes.Buffer
	printer.Fprint(&buf, fset, e)
	out := buf.String()

	// remove all white spaces
	out = removeAllWhiteSpaces(out)
	srcOut = removeAllWhiteSpaces(out)

	if out != srcOut {
		t.Errorf("Unexpected Output\n-- expected --\n%s\n-- was --\n%s\n", srcOut, out)
	}
}

func TestRelRegFile(t *testing.T) {
	srcOut := srcExpected

	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("could not get working directory")
		return
	}

	fun := fun("", "tmp.go")
	tmpFilePath := filepath.Join(wd, fun.File)
	f, err := os.OpenFile(tmpFilePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		t.Errorf("could not create file: %v", err)
		return
	}
	fmt.Printf("created temp file: %s\n", tmpFilePath)
	defer func() {
		// delete file
		err := os.Remove(tmpFilePath)
		if err != nil {
			fmt.Printf("could not delete file: %s\n", tmpFilePath)
		} else {
			fmt.Printf("deleted tmp file: %s\n", tmpFilePath)
		}
	}()

	_, err = f.Write([]byte(src))
	if err != nil {
		t.Errorf("could not write to file")
		return
	}
	err = f.Close()
	if err != nil {
		t.Errorf("could not close file")
	}

	ri := NewRelative(wd, 1.0)
	err = ri.Trans(fun)
	if err != nil {
		t.Errorf("could not transform file: %v", err)
		return
	}

	fc, err := ioutil.ReadFile(tmpFilePath)
	out := string(fc)

	// remove all whitespaces
	srcOut = removeAllWhiteSpaces(srcOut)
	out = removeAllWhiteSpaces(out)
	if out != srcOut {
		t.Errorf("Unexpected Output\n-- expected --\n%s\n-- was --\n%s\n", srcOut, out)
	}
}

func removeAllWhiteSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

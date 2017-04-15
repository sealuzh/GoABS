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

// helper functions
type srcFunc func() (string, string)

func testRelRegIntroFunc(srcFunc srcFunc, fun data.Function, t *testing.T) {
	src, srcOut := srcFunc()

	fset := token.NewFileSet()
	e, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		t.Errorf("Could not parse file: %v", err)
		return
	}

	visitor := &relRegVisitor{
		fun:       fun,
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

func testRelRegFile(srcFunc srcFunc, fun data.Function, t *testing.T) {
	src, srcOut := srcFunc()

	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("could not get working directory")
		return
	}

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

func fun(pkg, file, recv string) data.Function {
	return data.Function{
		Name:     "test",
		Pkg:      pkg,
		File:     file,
		Receiver: recv,
	}
}

//
// tests
//

// functions with return value tests

func funSrcReturn() (src, srcExpected string) {
	src = `
	package regression
	
	import "fmt"

	func test1() {}
	
	func test() string {
		fmt.Println("test func")
		return ""
	}

	func test2() {}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"

	func test1() {}
		
	func test() string {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
		return ""
	}

	func test2() {}
	`
	return
}

func TestRelRegStrReturn(t *testing.T) {
	fun := fun("", "", "")
	testRelRegIntroFunc(funSrcReturn, fun, t)
}
func TestRelRegFileReturn(t *testing.T) {
	fun := fun("", "tmp.go", "")
	testRelRegFile(funSrcReturn, fun, t)
}

// function with no (void) return value tests

func funSrcVoid() (src, srcExpected string) {
	src = `
	package regression
	
	import "fmt"

	func test() {
		fmt.Println("test func")
	}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"
		
	func test() {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
	}
	`
	return
}

func TestRelRegStrVoid(t *testing.T) {
	fun := fun("", "", "")
	testRelRegIntroFunc(funSrcVoid, fun, t)
}
func TestRelRegFileVoid(t *testing.T) {
	fun := fun("", "tmp.go", "")
	testRelRegFile(funSrcVoid, fun, t)
}

// methods (value receiver) with no return value test

func funSrcValueRecvVoid() (src, srcExpected string) {
	src = `
	package regression
	
	import "fmt"

	type T struct{}

	func (t T) test() {
		fmt.Println("test func")
	}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"

	type T struct{}
		
	func (t T) test() {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
	}
	`
	return
}

func TestRelRegStrValueRecvVoid(t *testing.T) {
	fun := fun("", "", "T")
	testRelRegIntroFunc(funSrcValueRecvVoid, fun, t)
}
func TestRelRegFileValueRecvVoid(t *testing.T) {
	fun := fun("", "tmp.go", "T")
	testRelRegFile(funSrcValueRecvVoid, fun, t)
}

// methods (value receiver) with return value test

func funSrcValueRecvReturn() (src, srcExpected string) {
	src = `
	package regression
	
	import "fmt"

	type T struct{}

	func (t T) test() string {
		fmt.Println("test func")
		return ""
	}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"

	type T struct{}
		
	func (t T) test() string {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
		return ""
	}
	`
	return
}

func TestRelRegStrValueRecvReturn(t *testing.T) {
	fun := fun("", "", "T")
	testRelRegIntroFunc(funSrcValueRecvReturn, fun, t)
}
func TestRelRegFileValueRecvReturn(t *testing.T) {
	fun := fun("", "tmp.go", "T")
	testRelRegFile(funSrcValueRecvReturn, fun, t)
}

// methods (pointer receiver) with no return value test

func funSrcPointerRecvVoid() (src, srcExpected string) {
	src = `
	package regression
	
	import "fmt"

	type T struct{}

	func (t *T) test() {
		fmt.Println("test func")
	}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"

	type T struct{}
		
	func (t *T) test() {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
	}
	`
	return
}

func TestRelRegStrPointerRecvVoid(t *testing.T) {
	fun := fun("", "", "*T")
	testRelRegIntroFunc(funSrcPointerRecvVoid, fun, t)
}
func TestRelRegFilePointerRecvVoid(t *testing.T) {
	fun := fun("", "tmp.go", "*T")
	testRelRegFile(funSrcPointerRecvVoid, fun, t)
}

// methods (pointer receiver) with return value test

func funSrcPointerRecvReturn() (src, srcExpected string) {
	src = `
	package regression
	
	import "fmt"

	type T struct{}

	func (t *T) test() string {
		fmt.Println("test func")
		return ""
	}
	`
	srcExpected = `
	package regression

	import "time"

	import "fmt"

	type T struct{}
		
	func (t *T) test() string {
		_goptcRegrStart := time.Now()
		fmt.Println("test func")
		time.Sleep(time.Duration(float32(time.Since(_goptcRegrStart).Nanoseconds()) * 1.000000))
		return ""
	}
	`
	return
}

func TestRelRegStrPointerRecvReturn(t *testing.T) {
	fun := fun("", "", "*T")
	testRelRegIntroFunc(funSrcPointerRecvReturn, fun, t)
}
func TestRelRegFilePointerRecvReturn(t *testing.T) {
	fun := fun("", "tmp.go", "*T")
	testRelRegFile(funSrcPointerRecvReturn, fun, t)
}

package count

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bitbucket.org/sealuzh/goptc/executil"
	"bitbucket.org/sealuzh/goptc/trans"
)

const (
	pkgName   = "ptcTraceWriter"
	fileName  = pkgName + ".go"
	writerVar = "PtcTraceWriter"

	godepsFolder = "Godeps"
)

func Functions(project, traceLibrary, out string, execTests bool) error {
	// create writer
	writerPkgName, err := createWriter(traceLibrary, out)
	if err != nil {
		fmt.Println("Could not create trace writer")
		return err
	}

	projectName := filepath.Base(project)
	libraryName := filepath.Base(traceLibrary)

	// transform traceLibrary
	err = transformLibrary(traceLibrary, writerPkgName, projectName, libraryName)
	if err != nil {
		fmt.Println("Could not transform library")
		return err
	}

	if execTests {
		// execute unit tests
		err = os.Chdir(project)
		if err != nil {
			fmt.Printf("Could not change directory to %s\n", project)
			return err
		}

		c := exec.Command("go", "test", "./...")
		c.Env = executil.Env(executil.GoPath(project))
		res, err := c.CombinedOutput()
		if err != nil {
			fmt.Printf("Error while executing go test command: %v\n", err)
			if res != nil {
				fmt.Println(res)
			}
			return err
		}
		fmt.Println(res)
	}
	return nil
}

func createWriter(traceLibrary, out string) (string, error) {
	pkgPath := filepath.Join(traceLibrary, pkgName)
	err := os.Mkdir(pkgPath, os.ModePerm)
	if err != nil {
		fmt.Printf("Could not create trace writer directory")
		return "", err
	}
	pathArr := strings.Split(traceLibrary, string(filepath.Separator))
	inVendor := false
	var buf bytes.Buffer
	for _, pe := range pathArr {
		if pe == "vendor" {
			inVendor = true
			continue
		}
		if inVendor {
			buf.WriteString(pe)
			buf.WriteString("/")
		}
	}
	basePkgName := buf.String()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("Could not retrieve working directory")
		return "", err
	}

	writerSrc := `
	package %s

	import (
		"io"
		"os"
		"fmt"
	)

	var %s = createTraceWriter("%s")

	func createTraceWriter(outPath string) io.Writer {
		f, err := os.Create(outPath)
		if err != nil {
			panic(fmt.Sprint("Could not create PTC trace writer file: " + err.Error()))
		}
		_, err = f.WriteString("LIB;PROJECT;METHOD")
		if err != nil {
			panic(fmt.Sprint("Could not write csv header to file"))
		}
		return f
	}
	`
	err = ioutil.WriteFile(filepath.Join(pkgPath, fileName),
		[]byte(fmt.Sprintf(writerSrc, pkgName, writerVar, filepath.Join(wd, out))),
		os.ModePerm)
	return filepath.Join(basePkgName, pkgName), err
}

func transformLibrary(path, writerPkgName, projectName, libraryName string) error {
	fmt.Printf("Start transforming library %s\n", path)
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			pathElems := strings.Split(path, string(filepath.Separator))
			for _, el := range pathElems {
				// remove all hidden folders
				if strings.HasPrefix(el, ".") || el == pkgName || el == godepsFolder {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			// not a go file
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			// do not transform test files
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return err
		}
		fmt.Printf("  transform file %s\n", path)

		err = transformFile(path, f, fset, writerPkgName, projectName, libraryName)
		if err != nil {
			fmt.Printf("Could not transform file %s\n", path)
			return err
		}

		return nil
	})
	return err
}

func transformFile(path string, f *ast.File, fset *token.FileSet, writerPkgName, projectName, libraryName string) error {
	v := publicFuncCountVisitor{
		writerPkgName: writerPkgName,
		projectName:   projectName,
		libraryName:   libraryName,
		relPath:       relPath(path, libraryName),
		transformed:   &transformed{},
	}

	ast.Walk(v, f)

	if v.transformed.v {
		// add import
		pkgNameRet := trans.AddImport(writerPkgName, f)
		if pkgName != pkgNameRet {
			// should never be the case
			// if it is the case that means that someone already used the package name in the imports
			panic(fmt.Sprintf("pkgName '%s' != pkgNameRet '%s'", pkgName, pkgNameRet))
		}

		// write transformed file back to file
		file, err := os.Create(path)
		if err != nil {
			fmt.Printf("Can not open file for rewriting it: %s\n", path)
			return err
		}
		err = printer.Fprint(file, fset, f)
		if err != nil {
			fmt.Printf("Can not write transformed src back to file: %s\n", path)
			return err
		}
	}

	return nil
}

func relPath(path, libraryName string) string {
	pathArr := strings.Split(path, string(filepath.Separator))
	inLib := false
	var buf bytes.Buffer
	for _, el := range pathArr {
		if el == libraryName {
			inLib = true
			continue
		}
		if inLib {
			buf.WriteString(el)
			buf.WriteString("/")
		}
	}
	return buf.String()
}

type transformed struct {
	v bool
}

type publicFuncCountVisitor struct {
	writerPkgName string
	projectName   string
	libraryName   string
	relPath       string
	transformed   *transformed
}

func (v publicFuncCountVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		return v.VisitFuncDecl(n)
	}
	return v
}

func (v publicFuncCountVisitor) VisitFuncDecl(node *ast.FuncDecl) ast.Visitor {
	funcName := node.Name.Name
	if !ast.IsExported(funcName) {
		// do not transform because this function/method is not part of the public API
		return v
	}

	var recv string
	rt, err := trans.ReceiverType(node)
	if err == nil {
		// is method
		recv = fmt.Sprintf("(%s).", rt)
	}

	write := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   ast.NewIdent(pkgName),
				Sel: ast.NewIdent(writerVar),
			},
			Sel: ast.NewIdent("Write"),
		},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun: ast.NewIdent("[]byte"),
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: fmt.Sprintf("\"%s;%s;%s\"", v.libraryName, v.projectName, filepath.Join(v.relPath, fmt.Sprintf("%s%s", recv, funcName))),
					},
				},
			},
		},
	}

	// add write to the start of the body
	list := make([]ast.Stmt, 0, len(node.Body.List)+1)
	list = append(list, &ast.ExprStmt{
		X: write,
	})
	list = append(list, node.Body.List...)
	node.Body.List = list

	v.transformed.v = true

	return v
}

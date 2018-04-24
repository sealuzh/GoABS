package regression

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sealuzh/goabs/data"
	"github.com/sealuzh/goabs/utils/astutil"
)

type Introducer interface {
	Trans(f data.Function) error
	Reset() error
}

type relIntroducer struct {
	basePath  string
	violation float32
}

func NewRelative(basePath string, violation float32) Introducer {
	return &relIntroducer{
		basePath:  basePath,
		violation: violation,
	}
}

func (i *relIntroducer) Trans(fun data.Function) error {
	filePath := filepath.Join(i.basePath, fun.Pkg, fun.File)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		fmt.Printf("Could not parse file: %s\n", filePath)
		return err
	}

	v := &relRegVisitor{
		fun:       fun,
		violation: i.violation,
	}

	ast.Walk(v, f)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		fmt.Printf("Could not open file: %s\n", filePath)
		return err
	}

	err = printer.Fprint(file, fset, f)
	if err != nil {
		file.Close()
		fmt.Printf("Could not save back to file: %s\n", filePath)
		return err
	}
	file.Close()

	return nil
}

func (i *relIntroducer) Reset() error {
	// save ast and restore it later
	return gitReset(i.basePath)
}

func gitReset(basePath string) error {
	err := os.Chdir(basePath)
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "reset", "--hard")
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Could not reset introduced regression with git")
		return err
	}
	return nil
}

type relRegVisitor struct {
	fun            data.Function
	violation      float32
	timeImportName string
}

func (v *relRegVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.File:
		return v.VisitFile(n)
	case *ast.FuncDecl:
		return v.VisitFuncDecl(n)
	}
	return v
}

func (v *relRegVisitor) VisitFile(node *ast.File) ast.Visitor {
	timePkgName := "time"
	importName := astutil.AddImport(timePkgName, node)
	v.timeImportName = importName
	return v
}

func (v *relRegVisitor) VisitFuncDecl(node *ast.FuncDecl) ast.Visitor {
	if !astutil.MatchingFunction(node, v.fun) {
		return v
	}

	newNodesCount := 2

	b := node.Body
	list := make([]ast.Stmt, 0, len(b.List)+newNodesCount)

	// time pkg selector
	time := ast.NewIdent(v.timeImportName)
	// start variable name
	startVarName := ast.NewIdent("_goptcRegrStart")
	// call to date.Now()
	dateNow := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent(v.timeImportName),
			Sel: ast.NewIdent("Now"),
		},
		Args: []ast.Expr{},
	}
	// assignment statement of start
	start := &ast.AssignStmt{
		Lhs: []ast.Expr{startVarName},
		Rhs: []ast.Expr{dateNow},
		Tok: token.DEFINE,
	}
	// add start to begin of method body
	list = append(list, start)

	// add deferred sleep
	// time.Sleep(time.Duration(float64(time.Since(start).Nanoseconds()) * 0.1))
	sleep := v.sleepStmt(time, startVarName)

	// wrap in immediately called closure of the form func() { sleep }()
	deferredSleep := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{&ast.ExprStmt{
						X: sleep,
					}},
				},
			},
		},
	}
	list = append(list, deferredSleep)

	list = append(list, b.List...)
	b.List = list
	return v
}

func (v *relRegVisitor) sleepStmt(timePkg, startVarName *ast.Ident) *ast.CallExpr {
	// duration
	dur := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   timePkg,
			Sel: ast.NewIdent("Since"),
		},
		Args: []ast.Expr{
			startVarName,
		},
	}

	// nanoseconds of duration
	ns := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   dur,
			Sel: ast.NewIdent("Nanoseconds"),
		},
		Args: []ast.Expr{},
	}

	// nanoseconds in float32
	fns := &ast.CallExpr{
		Fun: ast.NewIdent("float32"),
		Args: []ast.Expr{
			ns,
		},
	}

	// sleep time
	sleepTime := &ast.BinaryExpr{
		X:  fns,
		Op: token.MUL,
		Y: &ast.BasicLit{
			Value: fmt.Sprintf("%f", v.violation),
			Kind:  token.FLOAT,
		},
	}

	// duration of sleep time
	sleepDuration := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   timePkg,
			Sel: ast.NewIdent("Duration"),
		},
		Args: []ast.Expr{
			sleepTime,
		},
	}

	// sleep statement
	sleep := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   timePkg,
			Sel: ast.NewIdent("Sleep"),
		},
		Args: []ast.Expr{
			sleepDuration,
		},
	}

	return sleep
}

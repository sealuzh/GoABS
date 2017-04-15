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

	"bitbucket.org/sealuzh/goptc/data"
	"bitbucket.org/sealuzh/goptc/trans"
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
	timePath := fmt.Sprintf("\"%s\"", timePkgName)
	var timeImported bool
	for _, is := range node.Imports {
		if is.Path.Value == timePath {
			timeImported = true
			if is.Name != nil {
				v.timeImportName = is.Name.Name
			}
		}
	}
	if !timeImported {
		is := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: timePath,
			},
		}
		newDecls := make([]ast.Decl, 0, len(node.Decls)+1)
		newDecls = append(newDecls, &ast.GenDecl{
			Specs: []ast.Spec{is},
			Tok:   token.IMPORT,
		})
		newDecls = append(newDecls, node.Decls...)
		node.Decls = newDecls
		node.Imports = append(node.Imports, is)
		v.timeImportName = timePkgName
	}

	return v
}

func (v *relRegVisitor) VisitFuncDecl(node *ast.FuncDecl) ast.Visitor {
	if !trans.MatchingFunction(node, v.fun) {
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
	list = append(list, b.List...)

	// add actual regression
	// time.Sleep(time.Duration(float64(time.Since(start).Nanoseconds()) * 0.1))
	sleep := v.sleepStmt(time, startVarName)
	// add sleep to end of body
	lastStmt := b.List[len(b.List)-1]
	list = append(list, sleep)
	lenList := len(list)
	switch lastStmt.(type) {
	case *ast.ReturnStmt:
		el := list[lenList-1]
		list[lenList-1] = list[lenList-2]
		list[lenList-2] = el
	}

	b.List = list
	return v
}

func (v *relRegVisitor) sleepStmt(timePkg, startVarName *ast.Ident) *ast.ExprStmt {
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

	return &ast.ExprStmt{
		X: sleep,
	}
}

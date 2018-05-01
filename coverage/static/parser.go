package static

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/loader"

	"github.com/sealuzh/goabs/data"
	"github.com/sealuzh/goabs/utils/astutil"
)

const (
	builtin         = "builtin"
	pkgLevelCaller  = "<pkg_level_caller>"
	missingTypeInfo = "!missing_type_info!"
)

type MissingTypeInformation string

func (e MissingTypeInformation) Error() string {
	return string(e)
}

func ParseFiles(pkg string, fs []*ast.File, info *loader.PackageInfo) (map[data.Function][]data.Function, []error) {
	errs := []error{}
	callsites := map[data.Function][]data.Function{}

	for _, f := range fs {
		fn := "" // bo file name information available
		v := newFileVisitor(pkg, fn, info)
		ast.Walk(v, f)
		if len(v.errs) > 0 {
			//fmt.Printf("Do not continue walking files: %v\n", v.errs)
			errs = append(errs, v.errs...)
			//break Loop
		}

		addCallsites(callsites, v.callsites)
	}

	return callsites, errs
}

func addCallsites(m map[data.Function][]data.Function, n map[data.Function][]data.Function) {
	for f, cs := range n {
		// check if callsites for function exist -> should not be the case
		_, ok := m[f]
		if !ok {
			m[f] = []data.Function{}
		}
		m[f] = append(m[f], cs...)
	}
}

func newFileVisitor(pkg string, file string, info *loader.PackageInfo) *fileVisitor {
	return &fileVisitor{
		pkg:       pkg,
		file:      file,
		errs:      []error{},
		info:      info,
		callsites: map[data.Function][]data.Function{},
	}
}

type fileVisitor struct {
	pkg       string
	file      string
	errs      []error
	info      *loader.PackageInfo
	callsites map[data.Function][]data.Function
}

func (v *fileVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		// within-function call
		return v.VisitFuncDecl(n)
	case *ast.CallExpr:
		// package-level call
		return v.VisitCallExpr(n)
	default:
	}
	return v
}

func (v *fileVisitor) addCallsitesAndErrors(fun data.Function, csv *callSiteVisitor) {
	// append errors if existing
	if len(csv.errs) > 0 {
		//TODO: add function information to errors
		v.errs = append(v.errs, csv.errs...)
	}

	// add callsites
	_, ok := v.callsites[fun]
	if !ok {
		// no callsites yet
		v.callsites[fun] = []data.Function{}
	}
	// add callsites
	v.callsites[fun] = append(v.callsites[fun], csv.cs...)
}

func (v *fileVisitor) VisitFuncDecl(f *ast.FuncDecl) ast.Visitor {
	n := f.Name.Name
	recv, _ := astutil.UntypedReceiverType(f)
	fun := data.Function{
		Receiver:  recv,
		Name:      n,
		Pkg:       v.pkg,
		File:      v.file,
		StartLine: int(f.Pos()),
		EndLine:   int(f.End()),
	}

	csv := newCallSiteVisitor(v.info, fun)

	ast.Walk(csv, f.Body)

	v.addCallsitesAndErrors(fun, csv)

	return nil
}

func (v *fileVisitor) VisitCallExpr(n *ast.CallExpr) ast.Visitor {
	fun := data.Function{
		Pkg:       v.pkg,
		File:      v.file,
		Name:      pkgLevelCaller,
		StartLine: int(n.Pos()),
		EndLine:   int(n.End()),
	}

	csv := newCallSiteVisitor(v.info, fun)

	ast.Walk(csv, n)

	v.addCallsitesAndErrors(fun, csv)

	return nil
}

func newCallSiteVisitor(info *loader.PackageInfo, fun data.Function) *callSiteVisitor {
	return &callSiteVisitor{
		info: info,
		fun:  fun,
		cs:   []data.Function{},
		errs: []error{},
	}
}

type callSiteVisitor struct {
	info *loader.PackageInfo
	fun  data.Function
	cs   []data.Function
	errs []error
}

func (v *callSiteVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CallExpr:
		v.VisitCallExpr(n)
	}
	return v
}

func (v *callSiteVisitor) VisitCallExpr(n *ast.CallExpr) ast.Visitor {
	var callee data.Function
	var err error

	switch f := n.Fun.(type) {
	case *ast.SelectorExpr:
		// function of other package or method of a type
		callee, err = v.fqn(f.Sel)
	case *ast.Ident:
		// either function within same package or built-in function
		callee, err = v.fqn(f)
	case *ast.FuncLit:
		// anonymous function -> same scope as enclosing function -> no special handling
		return v
	default:
		return v
	}

	if err != nil {
		// do not add errors regarding non-function calls (e.g., type asserts)
		//v.errs = append(v.errs, err)
		if _, ok := err.(MissingTypeInformation); ok {
			// append calls with missing type information
			v.cs = append(v.cs, callee)
		}
	} else {
		v.cs = append(v.cs, callee)
	}

	return v
}

func nfqn(t string, pkg string) (string, error) {
	if strings.Contains(t, pkg) {
		// remove package
		removedPkg := strings.Replace(t, pkg, "", -1)
		removedDots := strings.Replace(removedPkg, ".", "", -1)
		return removedDots, nil
	}
	return t, fmt.Errorf("Type '%s' not of package '%s'", t, pkg)
}

func (v *callSiteVisitor) fqn(funcName *ast.Ident) (data.Function, error) {
	o := v.info.ObjectOf(funcName)
	if o == nil {
		return data.Function{
			Pkg:  missingTypeInfo,
			Name: funcName.Name,
		}, MissingTypeInformation(fmt.Sprintf("For %s", funcName.Name))
	}

	fun := data.Function{
		Name: o.Name(),
	}

	var valid bool

	switch f := o.(type) {
	case *types.Builtin:
		fun.Pkg = builtin
		valid = true
	case *types.Func:
		if pkg := f.Pkg(); pkg != nil {
			fun.Pkg = pkg.Path()
		} else {
			fun.Pkg = builtin
		}

		s := f.Type().(*types.Signature)
		if r := s.Recv(); r != nil {
			// has receiver -> is method on type
			if t := r.Type(); t != nil {
				// type information for receiver available
				fun.Receiver, _ = nfqn(t.String(), fun.Pkg)
			} else {
				// no type information for receiver available
				fun.Receiver = missingTypeInfo
			}
		}
		valid = true
	case *types.TypeName:
		// do not handle type asserts
		valid = false
	default:
		// all other object types (*types.Var, *types.Const, *types.Label, *types.PkgName, and *types.Nil) should not occur
		valid = false
	}

	if !valid {
		return fun, fmt.Errorf("Not a function/method call")
	}

	return fun, nil
}

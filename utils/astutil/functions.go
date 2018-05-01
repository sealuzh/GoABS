package astutil

import (
	"fmt"
	"go/ast"

	"github.com/sealuzh/goabs/data"
)

func ReceiverType(fn *ast.FuncDecl, pkg string) (string, error) {
	if fn.Recv == nil {
		// function and not method
		return "", fmt.Errorf("%s is not a method", fn.Name.Name)
	}
	switch e := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return typed(pkg, e), nil
	case *ast.StarExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			return fmt.Sprintf("*%s", typed(pkg, id)), nil
		}
	}
	// The parser accepts much more than just the legal forms.
	return "", fmt.Errorf("Invalid receiver type for %s", fn.Name.Name)
}

func typed(pkg string, ident *ast.Ident) string {
	if pkg == "" {
		return ident.Name
	}
	return fmt.Sprintf("%s.%s", pkg, ident.Name)
}

func UntypedReceiverType(fn *ast.FuncDecl) (string, error) {
	return ReceiverType(fn, "")
}

func MatchingFunction(node *ast.FuncDecl, fun data.Function) bool {
	// name match
	match := node.Name.Name == fun.Name
	// receiver match
	if node.Recv != nil {
		// method
		typeName, err := UntypedReceiverType(node)
		if err != nil {
			fmt.Println(err)
			return false
		}
		match = match && typeName == fun.Receiver
	} else {
		// function
		match = match && fun.Receiver == ""
	}
	// no parameter/return type matching necessary as Go does not provide Function-overloading
	return match
}

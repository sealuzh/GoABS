package trans

import (
	"fmt"
	"go/ast"

	"bitbucket.org/sealuzh/goptc/data"
)

func MatchingFunction(node *ast.FuncDecl, fun data.Function) bool {
	// name match
	match := node.Name.Name == fun.Name
	// receiver match
	if node.Recv != nil {
		// method
		typeName, err := ReceiverType(node)
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

func ReceiverType(fn *ast.FuncDecl) (string, error) {
	if fn.Recv == nil {
		// function and not method
		return "", fmt.Errorf("%s is not a method", fn.Name.Name)
	}
	switch e := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return e.Name, nil
	case *ast.StarExpr:
		if id, ok := e.X.(*ast.Ident); ok {
			return fmt.Sprintf("*%s", id.Name), nil
		}
	}
	// The parser accepts much more than just the legal forms.
	return "", fmt.Errorf("Invalid receiver type for %s", fn.Name.Name)
}

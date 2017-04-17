package trans

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func AddImport(importName string, node *ast.File) string {
	var imported bool
	importPath := fmt.Sprintf("\"%s\"", importName)
	for _, is := range node.Imports {
		if is.Path.Value == importPath {
			imported = true
			if is.Name != nil {
				importName = is.Name.Name
			}
		}
	}
	if !imported {
		is := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: importPath,
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

		// find last / in import name
		posSlash := strings.LastIndex(importName, "/")
		if posSlash >= 0 {
			importName = importName[posSlash+1 : len(importName)]
		}
	}

	return importName
}

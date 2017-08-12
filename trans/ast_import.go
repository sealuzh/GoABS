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

		li := lastImportStmt(node.Decls)

		newDecls := make([]ast.Decl, 0, len(node.Decls)+1)
		newDecls = append(newDecls, node.Decls[:li]...)
		newDecls = append(newDecls, &ast.GenDecl{
			Specs: []ast.Spec{is},
			Tok:   token.IMPORT,
		})
		newDecls = append(newDecls, node.Decls[li:]...)
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

func lastImportStmt(decls []ast.Decl) int {
	i := 0
	inImports := false
L:
	for ii, decl := range decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				inImports = true
			} else if inImports {
				i = ii
				break L
			}
		default:
			if inImports {
				i = ii
				break L
			}
		}
	}
	return i
}

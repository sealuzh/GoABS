package bench

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"bitbucket.org/sealuzh/goptc/data"
)

const (
	depFolder        = "vendor"
	goTestFileSuffix = "_test.go"
	benchFuncPrefix  = "Benchmark"

	defaultBenchCount = 5
)

func Functions(rootPath string) (data.PackageMap, error) {
	paths := make(data.PackageMap)
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		if !isValidDir(path) {
			return filepath.SkipDir
		}

		pkg := strings.Replace(path, rootPath, "", -1)

		fileInfos, err := ioutil.ReadDir(path)
		if err != nil {
			return err
		}

		for _, fi := range fileInfos {
			if fi.IsDir() {
				continue
			}
			fn := fi.Name()
			if strings.HasSuffix(fn, goTestFileSuffix) {
				benchs, err := parseFile(filepath.Join(path, fn), pkg, fn)
				if err != nil {
					return err
				}

				if len(benchs) > 0 {
					p, ok := paths[pkg]
					// pkg exists in out?
					if ok {
						_, ok := p[fn]
						if ok {
							fmt.Printf("ERROR - file (%s) in package (%s) already exists\n", fn, pkg)
						}
						// set/overwrite benchmarks of file
						p[fn] = benchs
					} else {
						paths[pkg] = make(map[string]data.File)
						paths[pkg][fn] = benchs
					}
				}
			}
		}
		return nil
	})

	return paths, err
}

func parseFile(path, pkg, fn string) (data.File, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	benchs := make([]data.Function, 0, defaultBenchCount)

	v := &BenchVisitor{
		path: pkg,
		fn:   fn,
		bs:   benchs,
	}
	ast.Walk(v, f)

	return v.bs, nil
}

func isValidDir(path string) bool {
	pathElems := strings.Split(path, string(filepath.Separator))

	for _, el := range pathElems {
		// remove everything from dependencies folder
		if el == depFolder {
			return false
		}
		// remove all hidden folders
		if strings.HasPrefix(el, ".") {
			return false
		}
	}

	return true
}

type BenchVisitor struct {
	path string
	fn   string
	bs   []data.Function
}

func (v *BenchVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		v.handleFuncDecl(n)
	}
	return v
}

func (v *BenchVisitor) handleFuncDecl(f *ast.FuncDecl) {
	n := f.Name.Name
	if !strings.HasPrefix(n, benchFuncPrefix) {
		return
	}

	fun := data.Function{
		Path: v.path,
		File: v.fn,
		Name: n,
	}
	v.bs = append(v.bs, fun)
}

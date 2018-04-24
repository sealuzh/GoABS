package bench

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sealuzh/goabs/data"
	"github.com/sealuzh/goabs/util"
)

const (
	goTestFileSuffix = "_test.go"
	benchFuncPrefix  = "Benchmark"

	defaultBenchCount = 5
)

func MatchingFunctions(rootPath, benchRegex string) (data.PackageMap, error) {
	paths := make(data.PackageMap)
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		if !util.IsValidDir(path) {
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
				benchs, err := parseFile(filepath.Join(path, fn), pkg, fn, benchRegex)
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

func Functions(rootPath string) (data.PackageMap, error) {
	return MatchingFunctions(rootPath, "^.*$")
}

func parseFile(path, pkg, fn, benchRegex string) (data.File, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	benchs := make([]data.Function, 0, defaultBenchCount)

	regex, err := regexp.Compile(benchRegex)
	if err != nil {
		return nil, err
	}

	v := &BenchVisitor{
		fset:       fset,
		pkg:        pkg,
		fn:         fn,
		bs:         benchs,
		benchRegex: regex,
	}
	ast.Walk(v, f)

	return v.bs, nil
}

type BenchVisitor struct {
	fset       *token.FileSet
	pkg        string
	fn         string
	bs         []data.Function
	benchRegex *regexp.Regexp
}

func (v *BenchVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		v.VisitFuncDecl(n)
	}
	return v
}

func (v *BenchVisitor) VisitFuncDecl(f *ast.FuncDecl) {
	n := f.Name.Name
	if !strings.HasPrefix(n, benchFuncPrefix) || !v.benchRegex.Match([]byte(n)) {
		return
	}

	start := v.fset.Position(f.Pos()).Line
	end := v.fset.Position(f.End()).Line

	fun := data.Function{
		Pkg:       v.pkg,
		File:      v.fn,
		Name:      n,
		StartLine: start,
		EndLine:   end,
	}
	v.bs = append(v.bs, fun)
}

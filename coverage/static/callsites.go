package static

import (
	"fmt"
	"go/build"
	"os"

	"github.com/sealuzh/goabs/coverage/callsite"
	"github.com/sealuzh/goabs/data"
	"golang.org/x/tools/go/loader"
)

const (
	defaultErrors = 10
)

var _ callsite.Finder = &staticCallSiteFinder{}

func NewStaticCallSiteFinder(path string, gopath string, recursivePackages bool, excludeTests bool) *staticCallSiteFinder {
	pkgs := callsite.Packages(path, gopath, recursivePackages)
	pkgsMap := map[string]struct{}{}
	for _, pkg := range pkgs {
		pkgsMap[pkg] = struct{}{}
	}

	return &staticCallSiteFinder{
		cs:           map[string]callsite.List{},
		csCounts:     map[string]int{},
		gopath:       gopath,
		pkgs:         pkgs,
		pkgsMap:      pkgsMap,
		excludeTests: excludeTests,
	}
}

type staticCallSiteFinder struct {
	parsed       bool
	count        int
	cs           map[string]callsite.List // call sites of a package; index is file path (from src on)
	csCounts     map[string]int
	csCountTotal int
	gopath       string
	pkgs         []string
	pkgsMap      map[string]struct{}
	excludeTests bool
}

func (f staticCallSiteFinder) All() (callsite.List, error) {
	if !f.parsed {
		return callsite.List{}, callsite.NotParsedError
	}
	ret := callsite.List(make([]callsite.Element, 0, f.count))
	for _, css := range f.cs {
		for _, cs := range css {
			ret = append(ret, cs)
		}
	}
	return ret, nil
}

func (f staticCallSiteFinder) Package(path string) (callsite.List, error) {
	if !f.parsed {
		return callsite.List{}, callsite.NotParsedError
	}

	fcss, ok := f.cs[path]
	if !ok {
		return callsite.List{}, callsite.PkgNotFoundError
	}

	ret := callsite.List(make([]callsite.Element, 0, len(fcss)))
	for _, cs := range fcss {
		ret = append(ret, cs)
	}
	return ret, nil
}

func (f *staticCallSiteFinder) addCallsiteCount(fun data.Function, count int) {
	pkg := fun.Pkg
	f.csCountTotal += count
	f.csCounts[pkg] += count
}

func (f *staticCallSiteFinder) addCallsites(callsites map[data.Function][]data.Function) {
	for caller, callsites := range callsites {
		pkg := caller.Pkg
		_, ok := f.cs[pkg]
		// create callsite.List if non-existing
		if !ok {
			f.cs[pkg] = callsite.List{}
		}

		callerCsCount := len(callsites)
		// add callsite count for pkg
		f.addCallsiteCount(caller, callerCsCount)

		for _, cs := range callsites {
			f.cs[pkg] = append(f.cs[pkg], callsite.Element{
				Caller: caller,
				Callee: cs,
			})
		}
	}
}

func (f *staticCallSiteFinder) Parse() error {
	var conf loader.Config
	conf.AllowErrors = true
	conf.Build = &build.Default
	conf.Build.GOPATH = f.gopath
	//conf.Build.UseAllFiles = true // adds problems for type resolver
	//conf.CreateFromFilenames(filepath.Join(gopath, pkg), fileName)
	for _, pkg := range f.pkgs {
		if f.excludeTests {
			conf.Import(pkg)
		} else {
			conf.ImportWithTests(pkg)
		}
	}

	lp, err := conf.Load()
	if err != nil {
		return err
	}

	// p := ssautil.CreateProgram(lp, ssa.BuilderMode(0))

	// for _, pkg := range p.AllPackages() {
	// 	fmt.Println(pkg.String())
	// 	pkgStr := strings.Replace(pkg.String(), "package ", "", -1)
	// 	if _, ok := f.pkgsMap[pkgStr]; ok {
	// 		pkg.Build()
	// 		f.parsePackage(pkg)
	// 	}
	// }

	errorFree := 0
	errors := 0
Loop:
	for _, pkg := range lp.AllPackages {
		pkgName := pkg.Pkg.Path()
		if _, ok := f.pkgsMap[pkgName]; ok {
			callsites, errs := ParseFiles(pkgName, pkg.Files, pkg)
			if pkg.TransitivelyErrorFree {
				errorFree++
			} else {
				errors++
			}
			if errs != nil && len(errs) > 0 {
				fmt.Fprintf(os.Stderr, "%v\n", errs)
				break Loop
			}
			f.addCallsites(callsites)
			//fmt.Printf("%v\n%v\n%t\ncs count: %d\n\n", t, pkg, pkg.TransitivelyErrorFree, f.csCounts[pkgName])
		}
	}
	//fmt.Printf("error-free: %d\nerrors: %d\n", errorFree, errors)

	f.parsed = true

	return nil
}

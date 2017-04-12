package bench

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/sealuzh/goptc/data"
)

const (
	goPathVariable = "GOPATH"
	srcFolder      = "src"

	cmdName        = "go"
	cmdArgsTest    = "test"
	cmdArgsBench   = "-bench=^%s$"
	cmdArgsCount   = "-count=%d"
	cmdArgsNoTests = "-run=^$"
	cmdArgsTimeout = "-timeout=%s"

	benchResultUnit = "ns/op"
	benchRuntime    = 1
	benchTimeoutMsg = "*** Test killed with quit: ran too long"
)

type Runner interface {
	Run(run int, test string) (int, error)
}

func NewRunner(projectRoot string, wi int, mi int, timeout string, out csv.Writer) (Runner, error) {
	pkgs, err := Functions(projectRoot)
	if err != nil {
		return nil, err
	}

	cmdCount := fmt.Sprintf(cmdArgsCount, (wi + mi))

	return &runnerWithPenalty{
		defaultRunner: defaultRunner{
			projectRoot: projectRoot,
			wi:          wi,
			mi:          mi,
			out:         out,
			benchs:      pkgs,
			env:         env(goPath(projectRoot)),
			cmdCount:    cmdCount,
			cmdArgs:     []string{cmdArgsTest, fmt.Sprintf(cmdArgsTimeout, timeout), cmdCount, cmdArgsNoTests},
		},
		penalisedBenchs: make(map[string]struct{}),
		timeout:         timeout,
	}, nil
}

type defaultRunner struct {
	projectRoot string
	wi          int
	mi          int
	out         csv.Writer
	benchs      data.PackageMap
	env         []string
	cmdCount    string
	cmdArgs     []string
}

type runnerWithPenalty struct {
	defaultRunner
	timeout         string
	penalisedBenchs map[string]struct{}
}

func (r *runnerWithPenalty) Run(run int, test string) (int, error) {
	benchCount := 0

	for pkgName, pkg := range r.benchs {
		fmt.Printf("# Execute Benchmarks in Dir: %s\n", pkgName)

		dir := filepath.Join(r.projectRoot, pkgName)

		err := os.Chdir(dir)
		if err != nil {
			return benchCount, err
		}

		for fileName, file := range pkg {
			fmt.Printf("## Execute Benchmarks of File: %s\n", fileName)
			for _, bench := range file {
				relBenchName := fmt.Sprintf("%s/%s::%s", pkgName, fileName, bench.Name)
				// check if benchmark is penaltised
				_, penaltised := r.penalisedBenchs[relBenchName]
				if penaltised {
					fmt.Printf("### Do not execute Benchmark due to penalty: %s\n", relBenchName)
					continue
				}

				fmt.Printf("### Execute Benchmark: %s\n", bench.Name)
				args := append(r.cmdArgs, fmt.Sprintf(cmdArgsBench, bench.Name))
				c := exec.Command(cmdName, args...)
				c.Env = r.env

				res, err := c.CombinedOutput()
				resStr := string(res)
				if err != nil {
					fmt.Printf("Error while executing command '%s\n", c.Args)
					if strings.Contains(resStr, benchTimeoutMsg) {
						fmt.Printf("%s timed out after %s\n", relBenchName, r.timeout)
						r.penalisedBenchs[relBenchName] = struct{}{}
						err = nil
						continue
					} else {
						fmt.Printf("%s\n", resStr)
					}
				}

				parsed, err := parseAndSaveBenchOut(test, run, bench, pkgName, resStr, r.out)
				if err != nil {
					return benchCount, err
				}
				if parsed {
					benchCount++
				}
			}
		}
	}
	return benchCount, nil
}

func TimedRun(r Runner, run int, test string) (int, error, time.Duration) {
	now := time.Now()
	execBenchs, err := r.Run(run, test)
	dur := time.Since(now)
	return execBenchs, err, dur
}

func env(goPath string) []string {
	env := os.Environ()
	ret := make([]string, 0, len(env)+1)
	added := false
	goPathDecl := fmt.Sprintf("%s=%s", goPathVariable, goPath)
	for _, e := range env {
		if strings.HasPrefix(e, goPathVariable) {
			ret = append(ret, goPathDecl)
			added = true
		} else {
			ret = append(ret, e)
		}
	}
	if !added {
		ret = append(ret, goPathDecl)
	}
	return ret
}

func parseAndSaveBenchOut(test string, run int, b data.Function, pkg string, res string, out csv.Writer) (bool, error) {
	resArr := strings.Fields(res)
	var parsed bool
	for i, f := range resArr {
		if f == benchResultUnit {
			parsed = true
			out.Write([]string{strconv.FormatInt(int64(run), 10),
				test,
				filepath.Join(pkg, b.File, b.Name),
				resArr[i-1],
			})
		}
	}
	out.Flush()
	return parsed, nil
}

func goPath(p string) string {
	pathArr := strings.Split(p, string(filepath.Separator))
	var c int
	for i, el := range pathArr {
		if el == srcFolder {
			c = i
			break
		}
	}
	return fmt.Sprintf("/%s", filepath.Join(pathArr[:c]...))
}

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
	"bitbucket.org/sealuzh/goptc/executil"
)

const (
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

// NewRunner creates a new benchmark runner.
// By default it returns a penalised runner that in consecutive runs only executes successful benchmark executions.
func NewRunner(projectRoot string, benchs data.PackageMap, wi int, mi int, timeout string, duration time.Duration, out csv.Writer) (Runner, error) {
	// if benchmark gets executed over time period, do not do warm-up iterations
	if duration > 0 {
		wi = 0
	}

	// if no measurement iterations are provided, always do at least one measurement
	if mi == 0 {
		mi = 1
	}

	cmdCount := fmt.Sprintf(cmdArgsCount, (wi + mi))

	return &runnerWithPenalty{
		defaultRunner: defaultRunner{
			projectRoot: projectRoot,
			wi:          wi,
			mi:          mi,
			duration:    duration,
			out:         out,
			benchs:      benchs,
			env:         executil.Env(executil.GoPath(projectRoot)),
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
	duration    time.Duration
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

func (r *runnerWithPenalty) RunBenchmark(bench data.Function, run int, test, pkgName, fileName string) (bool, error) {
	relBenchName := fmt.Sprintf("%s/%s::%s", pkgName, fileName, bench.Name)
	// check if benchmark is penaltised
	_, penaltised := r.penalisedBenchs[relBenchName]
	if penaltised {
		fmt.Printf("### Do not execute Benchmark due to penalty: %s\n", relBenchName)
		return false, nil
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
			return false, nil
		}
		fmt.Printf("%s\n", resStr)
	}

	parsed, err := parseAndSaveBenchOut(test, run, bench, pkgName, resStr, r.out)
	if err != nil {
		return false, err
	}
	if !parsed {
		fmt.Printf("%s result could not be parsed\n", relBenchName)
		r.penalisedBenchs[relBenchName] = struct{}{}
		return false, nil
	}
	return true, nil
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
				if r.duration == 0 {
					executed, err := r.RunBenchmark(bench, run, test, pkgName, fileName)
					if err != nil {
						return benchCount, err
					}
					if executed {
						benchCount++
					}
				} else {
					startBench := time.Now()
					for time.Since(startBench).Seconds() < r.duration.Seconds() {
						executed, err := r.RunBenchmark(bench, run, test, pkgName, fileName)
						if err != nil {
							return benchCount, err
						}
						if executed {
							// execution of benchmark was succsessful
							benchCount++
						} else {
							// execution of benchmark was not successful, do not execute it anymore
							break
						}
					}
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

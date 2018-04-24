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

	"github.com/sealuzh/goabs/data"
	"github.com/sealuzh/goabs/util"
)

const (
	cmdName           = "go"
	cmdArgsTest       = "test"
	cmdArgsBench      = "-bench=^%s$"
	cmdArgsCount      = "-count=%d"
	cmdArgsNoTests    = "-run=^$"
	cmdArgsTimeout    = "-timeout=%s"
	cmdArgsMem        = "-benchmem"
	cmdArgsProfileOut = "-outputdir=%s"
	cmdArgsCPUProfile = "-cpuprofile=%s"
	cmdArgsMemProfile = "-memprofile=%s"
	benchRuntime      = 1
	benchTimeoutMsg   = "*** Test killed with quit: ran too long"
)

type Runner interface {
	Run(run int, test string) (int, error)
}

// NewRunner creates a new benchmark runner.
// By default it returns a penalised runner that in consecutive runs only executes successful benchmark executions.
func NewRunner(projectRoot string, benchs data.PackageMap, wi int, mi int, timeout string, benchDuration time.Duration, runDuration time.Duration, benchMem bool, profile data.Profile, profileDir string, out csv.Writer) (Runner, error) {
	// if benchmark gets executed over time period, do not do warm-up iterations
	if benchDuration > 0 {
		wi = 0
	}

	// if no measurement iterations are provided, always do at least one measurement
	if mi == 0 {
		mi = 1
	}

	cmdCount := fmt.Sprintf(cmdArgsCount, (wi + mi))
	cmdArgs := []string{cmdArgsTest, fmt.Sprintf(cmdArgsTimeout, timeout), cmdCount, cmdArgsNoTests}

	var rp resultParser = rtResultParser{}
	if benchMem {
		cmdArgs = append(cmdArgs, cmdArgsMem)
		rp = memResultParser{}
	}

	return &runnerWithPenalty{
		defaultRunner: defaultRunner{
			projectRoot:   projectRoot,
			wi:            wi,
			mi:            mi,
			benchDuration: benchDuration,
			runDuration:   runDuration,
			benchMem:      benchMem,
			resultParser:  rp,
			out:           out,
			benchs:        benchs,
			profile:       profile,
			profileDir:    profileDir,
			env:           util.Env(util.GoPath(projectRoot)),
			cmdCount:      cmdCount,
			cmdArgs:       cmdArgs,
		},
		penalisedBenchs: make(map[string]struct{}),
		timeout:         timeout,
	}, nil
}

type defaultRunner struct {
	projectRoot   string
	wi            int
	mi            int
	benchDuration time.Duration
	benchMem      bool
	runDuration   time.Duration
	resultParser  resultParser
	out           csv.Writer
	benchs        data.PackageMap
	profile       data.Profile
	profileDir    string
	env           []string
	cmdCount      string
	cmdArgs       []string
}

type runnerWithPenalty struct {
	defaultRunner
	timeout         string
	penalisedBenchs map[string]struct{}
}

func (r *runnerWithPenalty) RunBenchmark(bench data.Function, run int, suiteExec int, test string) (int, error) {
	if r.benchDuration != 0 {
		startBench := time.Now()
		benchCount := 0
		for time.Since(startBench).Seconds() < r.benchDuration.Seconds() {
			exec, err := r.RunBenchmarkOnce(bench, run, suiteExec, benchCount, test)
			if err != nil || !exec {
				return benchCount, err
			}
			benchCount++
		}
		return benchCount, nil
	}

	// no benchmark duration supplied -> only one benchmark execution
	exec, err := r.RunBenchmarkOnce(bench, run, suiteExec, 0, test)
	if exec {
		return 1, err
	}
	return 0, err

}

func (r *runnerWithPenalty) RunBenchmarkOnce(bench data.Function, run int, suiteExec int, benchExec int, test string) (bool, error) {
	relBenchName := fmt.Sprintf("%s/%s::%s", bench.Pkg, bench.File, bench.Name)
	// check if benchmark is penaltised
	_, penaltised := r.penalisedBenchs[relBenchName]
	if penaltised {
		fmt.Printf("### Do not execute Benchmark due to penalty: %s\n", relBenchName)
		return false, nil
	}

	fmt.Printf("### Execute Benchmark: %s\n", bench.Name)
	args := append(r.cmdArgs, fmt.Sprintf(cmdArgsBench, bench.Name))
	// add profile if necessary
	if r.profile != data.NoProfile {
		args = r.profileCmdArgs(args, bench, run, suiteExec, benchExec, test)
	}

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

	result, err := r.resultParser.parse(resStr)
	if err != nil {
		if _, ok := err.(resultNotParsable); ok {
			fmt.Printf("%s result could not be parsed\n", relBenchName)
			r.penalisedBenchs[relBenchName] = struct{}{}
			return false, nil
		}
		return false, err
	}

	saveBenchOut(test, run, suiteExec, benchExec, bench, result, r.out, r.benchMem)

	return true, nil
}

func (r *runnerWithPenalty) RunUntil(run int, test string, done <-chan struct{}) (int, error) {
	benchCount := 0
Forever:
	for suiteExec := 0; true; suiteExec++ {
		for pkgName, pkg := range r.benchs {
			fmt.Printf("# Execute Benchmarks in Dir: %s\n", pkgName)

			dir := filepath.Join(r.projectRoot, pkgName)

			err := os.Chdir(dir)
			if err != nil {
				return benchCount, err
			}

			for fileName, file := range pkg {
				fmt.Printf("## Execute Benchmarks of File: %s\n", fileName)
			Bench:
				for _, bench := range file {
					executed, err := r.RunBenchmark(bench, run, suiteExec, test)

					if err != nil {
						return benchCount, err
					}
					benchCount += executed

					select {
					case <-done:
						break Forever
					default:
						continue Bench
					}
				}
			}
		}
	}
	return benchCount, nil
}

func (r *runnerWithPenalty) RunOnce(run int, test string) (int, error) {
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
				executed, err := r.RunBenchmark(bench, run, 0, test)
				if err != nil {
					return benchCount, err
				}
				benchCount += executed

			}
		}
	}
	return benchCount, nil
}

func (r *runnerWithPenalty) Run(run int, test string) (int, error) {
	if r.runDuration != 0 {
		done := make(chan struct{})
		go func() {
			select {
			case <-time.After(r.runDuration):
				close(done)
			}
		}()
		return r.RunUntil(run, test, done)
	}
	return r.RunOnce(run, test)
}

func (r *runnerWithPenalty) profileCmdArgs(args []string, bench data.Function, run int, suiteExec int, benchExec int, test string) []string {
	cmdProfileOut := fmt.Sprintf(cmdArgsProfileOut, r.profileDir)
	cpuPath := profileName(bench, run, suiteExec, benchExec, test, "cpu")
	cpuArg := fmt.Sprintf(cmdArgsCPUProfile, cpuPath)
	memPath := profileName(bench, run, suiteExec, benchExec, test, "mem")
	memArg := fmt.Sprintf(cmdArgsMemProfile, memPath)
	switch r.profile {
	case data.AllProfiles:
		args = append(args, cmdProfileOut, cpuArg, memArg)
	case data.CPUProfile:
		args = append(args, cmdProfileOut, cpuArg)
	case data.MemProfile:
		args = append(args, cmdProfileOut, memArg)
	}
	return args
}

func TimedRun(r Runner, run int, test string) (int, error, time.Duration) {
	now := time.Now()
	execBenchs, err := r.Run(run, test)
	dur := time.Since(now)
	return execBenchs, err, dur
}

func profileName(bench data.Function, run int, suiteExec int, benchExec int, test string, t string) string {
	return fmt.Sprintf("%d-%d-%d_%s_%s_%s_%s", run, suiteExec, benchExec, replaceSlashes(test), replaceSlashes(bench.Pkg), bench.Name, t)
}

func replaceSlashes(p string) string {
	if len(p) == 0 {
		return ""
	}

	// remove leading /
	if strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	if strings.HasSuffix(p, "/") {
		p = p[:len(p)-1]
	}
	return strings.Replace(p, "/", "-", -1)
}

func saveBenchOut(test string, run int, suiteExec int, benchExec int, b data.Function, res []result, out csv.Writer, benchMem bool) {
	outSize := 4
	if benchMem {
		outSize += 2
	}

	for _, result := range res {
		rec := make([]string, 0, outSize)
		rec = append(rec, fmt.Sprintf("%d-%d-%d", run, suiteExec, benchExec))
		rec = append(rec, test)
		rec = append(rec, filepath.Join(b.Pkg, b.File, b.Name))
		rec = append(rec, strconv.FormatFloat(float64(result.Runtime), 'f', -1, 32))

		if benchMem {
			rec = append(rec, strconv.FormatInt(int64(result.Memory), 10))
			rec = append(rec, strconv.FormatInt(int64(result.Allocations), 10))
		}

		out.Write(rec)
		out.Flush()
	}
}

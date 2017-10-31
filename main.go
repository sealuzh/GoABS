package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"bitbucket.org/sealuzh/goptc/bench"
	"bitbucket.org/sealuzh/goptc/data"
	"bitbucket.org/sealuzh/goptc/deps"
	"bitbucket.org/sealuzh/goptc/trans/count"
	"bitbucket.org/sealuzh/goptc/trans/regression"
)

const (
	defaultBenchTimeout = "10m"
)

// file (in and out) arguments
var configPath string
var out string

// operation flags
var dynamic bool
var trace bool

func parseArguments() {
	flag.StringVar(&configPath, "c", "", "config file")
	flag.StringVar(&out, "o", "", "output file")
	flag.BoolVar(&dynamic, "d", false, "dynamic coverage")
	flag.BoolVar(&trace, "t", false, "trace executions of public API")
	flag.Parse()
}

func parseConfig() data.Config {
	f, err := os.Open(configPath)
	if err != nil {
		panic(fmt.Errorf("Could not open config: %v", err))
	}

	var config data.Config
	d := json.NewDecoder(f)
	err = d.Decode(&config)
	if err != nil {
		panic(fmt.Errorf("Could not parse config file: %v", err))
	}

	// set defaults for config
	if config.DynamicConfig.Profile == "" {
		config.DynamicConfig.Profile = data.NoProfile
	}

	if config.DynamicConfig.Profile != data.NoProfile && config.DynamicConfig.ProfileDir == "" {
		panic(fmt.Errorf("No profile dir specified (-profileDir)"))
	}

	if config.DynamicConfig.ProfileDir != "" {
		fi, err := os.Stat(config.DynamicConfig.ProfileDir)
		if err != nil {
			panic(fmt.Sprintf("Profile directory error: %v", err))
		}
		if !fi.IsDir() {
			panic("Profile directory error: not a directory")
		}
	}

	return config
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	parseArguments()
	c := parseConfig()

	// install project dependencies
	if c.FetchDeps {
		err := deps.Fetch(c.Project)
		if err != nil {
			panic(err)
		}
	}

	if trace {
		err := count.Functions(c.Project, c.TraceLibrary, out, false)
		if err != nil {
			panic(err)
		}
		// when tracing every other operation can not be performed (due to config incompatibility)
		return
	}

	if dynamic {
		err := dptc(c)
		if err != nil {
			panic(err)
		}
	}
}

func dptc(c data.Config) error {
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	out := csv.NewWriter(f)
	out.Comma = ';'

	bto := c.DynamicConfig.BenchTimeout
	if bto == "" {
		bto = defaultBenchTimeout
	}

	var benchs data.PackageMap
	if benchRegex := c.DynamicConfig.BenchmarkRegex; benchRegex != "" {
		benchs, err = bench.MatchingFunctions(c.Project, benchRegex)
	} else {
		benchs, err = bench.Functions(c.Project)
	}

	if err != nil {
		return err
	}

	runner, err := bench.NewRunner(
		c.Project,
		benchs,
		c.DynamicConfig.WarmupIterations,
		c.DynamicConfig.MeasurementIterations,
		bto,
		time.Duration(c.DynamicConfig.BenchDuration),
		time.Duration(c.DynamicConfig.RunDuration),
		c.DynamicConfig.BenchMem,
		c.DynamicConfig.Profile,
		c.DynamicConfig.ProfileDir,
		*out,
	)
	if err != nil {
		return err
	}

	// check if function/method files can be opened
	err = checkFiles(c)
	if err != nil {
		fmt.Printf("Could not open one of the function/method files: %v\n", err)
		return err
	}

	runs := c.DynamicConfig.Runs
	if runs == 0 {
		runs = 1
	}

	clear := clearTmpFolder(c.ClearFolder)

	benchCounter := 0
	start := time.Now()
	regIntr := regression.NewRelative(c.Project, c.DynamicConfig.Regression)
	for run := 0; run < runs; run++ {
		fmt.Printf("---------- Run #%d ----------\n", run)
		// execute baseline run
		test := "Baseline"
		fmt.Printf("--- Run #%d of %s\n", run, test)
		execBenchs, err, dur := bench.TimedRun(runner, run, test)
		if err != nil {
			return err
		}
		fmt.Printf("--- Run #%d of %s executed %d which took %dns\n", run, test, execBenchs, dur.Nanoseconds())
		// clear tmp folder
		clear()

		benchCounter += execBenchs
		// execute benchmark suite with introduced regressions
		funs := c.DynamicConfig.Functions
		if c.DynamicConfig.Rmit {
			funs = rmitFuncs(c.DynamicConfig.Functions)
			fmt.Println("Using RMIT Methodology")
		}
		for _, f := range funs {
			test = f.String()
			fmt.Printf("--- Run #%d of %s\n", run, test)
			// introduce regression into function
			err := regIntr.Trans(f)
			if err != nil {
				fmt.Printf("Could not introduce regression into function %s\n", test)
				return err
			}
			execBenchs, err, dur := bench.TimedRun(runner, run, test)
			if err != nil {
				return err
			}
			fmt.Printf("--- Run #%d of %s executed %d which took %dns\n", run, test, execBenchs, dur.Nanoseconds())
			benchCounter += execBenchs

			// clear tmp folder
			clear()

			err = regIntr.Reset()
			if err != nil {
				fmt.Printf("Could not reset regression\n")
				return err
			}
		}
	}
	took := time.Since(start)
	fmt.Printf("\n%d Benchmarks executed in %d runs which took %dns\n", benchCounter, c.DynamicConfig.Runs, took.Nanoseconds())
	return nil
}

func clearTmpFolder(path string) func() {
	if path == "" {
		fmt.Println("No tmp folder to clear")
		return func() {}
	}
	// not closing file as we are using an anonymous function that uses that file
	folder, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("Could not open tmp folder: %v", err))
	}
	stat, err := folder.Stat()
	if err != nil {
		panic(fmt.Sprintf("Could not get info for folder: %v", err))
	}
	if !stat.IsDir() {
		panic(fmt.Sprintf("Path not a folder: %s", path))
	}
	folder.Close()
	return func() {
		contents, err := ioutil.ReadDir(path)
		if err != nil {
			// should not be the case
			fmt.Printf("Could not read dir: %s\n", path)
			return
		}
		for _, f := range contents {
			p := filepath.Join(path, f.Name())
			err := os.RemoveAll(p)
			if err != nil {
				fmt.Printf("Could not remove '%s'\n", p)
			}
		}
	}
}

func rmitFuncs(funcs []data.Function) []data.Function {
	l := len(funcs)
	ret := make([]data.Function, l)
	usedIx := make(map[int]struct{})
	rnd := rand.NewSource(time.Now().UnixNano())
	for _, f := range funcs {
		i := -1
		used := true
		for used {
			i = int(rnd.Int63()) % l
			_, used = usedIx[i]
		}

		if i < 0 {
			panic("i should not be below 0")
		}

		usedIx[i] = struct{}{}

		ret[i] = f
	}
	return ret
}

func checkFiles(c data.Config) error {
	for _, f := range c.DynamicConfig.Functions {
		path := filepath.Join(c.Project, f.Pkg, f.File)
		_, err := os.Stat(path)
		if err != nil {
			return err
		}
	}
	return nil
}

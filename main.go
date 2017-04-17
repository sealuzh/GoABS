package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"bitbucket.org/sealuzh/goptc/bench"
	"bitbucket.org/sealuzh/goptc/data"
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

	return config
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	parseArguments()
	c := parseConfig()

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

	bto := c.DynamicConfig.Timeout
	if bto == "" {
		bto = defaultBenchTimeout
	}

	runner, err := bench.NewRunner(
		c.Project,
		c.DynamicConfig.WarmupIterations,
		c.DynamicConfig.MeasurementIterations,
		bto,
		*out,
	)
	if err != nil {
		panic(err)
	}

	benchCounter := 0
	start := time.Now()
	regIntr := regression.NewRelative(c.Project, c.DynamicConfig.Regression)
	for run := 1; run <= c.DynamicConfig.Runs; run++ {
		fmt.Printf("---------- Run #%d ----------\n", run)
		// execute baseline run
		test := "baseline"
		fmt.Printf("--- Run #%d of %s\n", run, test)
		execBenchs, err, dur := bench.TimedRun(runner, run, test)
		if err != nil {
			return err
		}
		fmt.Printf("--- Run #%d of %s executed %d which took %dns\n", run, test, execBenchs, dur.Nanoseconds())
		benchCounter += execBenchs
		// execute benchmark suite with introduced regressions
		for _, f := range c.DynamicConfig.Functions {
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

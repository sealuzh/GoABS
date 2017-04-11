package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	"bitbucket.org/sealuzh/goptc/bench"
	"bitbucket.org/sealuzh/goptc/data"
)

var configPath string
var dynamic bool
var out string

func parseArguments() {
	flag.StringVar(&configPath, "c", "", "config file")
	flag.BoolVar(&dynamic, "d", false, "dynamic coverage")
	flag.StringVar(&out, "o", "", "output file")
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

	runner, err := bench.NewRunner(
		c.Project,
		c.DynamicConfig.WarmupIterations,
		c.DynamicConfig.MeasurementIterations,
		*out,
	)
	if err != nil {
		panic(err)
	}

	for run := 1; run <= c.DynamicConfig.Runs; run++ {
		// execute baseline run
		err = runner.Run(run, "baseline")
		if err != nil {
			return err
		}
		//TODO: introduce regression and execute benchmark
	}
	return nil
}

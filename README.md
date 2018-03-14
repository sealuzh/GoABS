# Go API Benchmarking Score (ABS)
GoABS is a tool to execute microbenchmarks written in Go.

Moreover, it is used in Laaber and Leitner's paper "An Evaluation of Open-Source Software Microbenchmark Suites for Continuous Performance Assessment" published at Mining Software Repositories (MSR) in 2018.

## Execution
Run the following script to execute ABS:
```bash
goabs -c gin.json -o gin_test_out.csv -d
```

### Arguments
* `-c` config file
* `-d` dynamic ABS metric
* `-o` output/result file

### Config File
Examplary configuration file for bleve project:
```json
{
	"project": "/home/ubuntu/bleve/src/github.com/blevesearch/bleve",
	"dynamic": {
		"bench_timeout": "3m",
		"i": 20,
		"runs": 2,
		"regression": 0.1,
		"functions": [
			{
				"pkg": "analysis",
				"file": "tokenmap.go",
				"name": "LoadLine",
				"recv": "TokenMap"
			},
			{
				"pkg": "index/upsidedown",
				"file": "row.go",
				"name": "NewDictionaryRow",
				"recv": ""
			}
		]
	}
}
```

JSON attributes (partial):
* `"project"` path to project directory
* `"dynamic"` settings related to Go benchmark execution and ABS
* `"i"` iterations/executions of each benchmark (uses `-count` flag of `go test`)
* `"runs"` complete experiment repeititions (r in MSR paper)
* `"regression"` relative slowdown introduced into functions 
* `"functions"` functions to inject regressions into (for ABS)

### Output
GoABS reports all results in CSV form to the file specified as `-o`.
A sample output file is depicted below:
```csv
Run-SuiteExecution-BenchmarkExecution?-?;Function altered;Benchmark;Runtime in ns 
0-0-0;Baseline;benchmarks_test.go/BenchmarkOneRoute;58.6
0-0-0;Baseline;benchmarks_test.go/BenchmarkOneRoute;60
0-0-0;Baseline;benchmarks_test.go/BenchmarkOneRoute;62.3
0-0-0;Baseline;benchmarks_test.go/BenchmarkOneRoute;61
0-0-0;Baseline;benchmarks_test.go/BenchmarkRecoveryMiddleware;107
0-0-0;Baseline;benchmarks_test.go/BenchmarkRecoveryMiddleware;112
0-0-0;Baseline;benchmarks_test.go/BenchmarkRecoveryMiddleware;112
0-0-0;Baseline;benchmarks_test.go/BenchmarkRecoveryMiddleware;123
```

Run gets increased according to json attribute `"runs"`, SuiteExecution according to `"run_duration"`, and BenchmarkExecution according to `"bench_duration"`. Intuitively, `"runs"` defines how often the benchmark suite should be executed, `"run_duration"` defines how long each suite is executed (potentially multiple times), and `"bench_duration"` defines how long each benchmark is executed (potentially multiple times). All values start at 0.

## Tracing of API Asage

### Execution
```bash
goabs -c config.json -t -o trace_out.csv
cd PATH/TO/UNIT_TEST_LIB
go test ./...
```

### Config File
```json
{
	"project":  "PATH/TO/UNIT_TEST_LIB",
	"trace_lib": "PATH/TO/API_TRACE_LIB"
}
```

Use trace aggregator of [JavaAPIUsageTracer](https://github.com/sealuzh/JavaAPIUsageTracer) to sum traces for each function.

Remark: do not forget to set the GOPATH correctly, and retrieve the dependencies og the unit test library before running script.


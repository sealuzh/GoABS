package data

type Config struct {
	Project       string        `json:"project"`
	DynamicConfig DynamicConfig `json:"dynamic"`
	TraceLibrary  string        `json:"trace_lib"`
}

type DynamicConfig struct {
	WarmupIterations      int        `json:"wi"`
	MeasurementIterations int        `json:"i"`
	Runs                  int        `json:"runs"`
	Timeout               string     `json:"bench_timeout"`
	Regression            float32    `json:"regression"`
	Functions             []Function `json:"functions"`
	Rmit                  bool       `json:"rmit"`
}

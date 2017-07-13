package data

import "time"

type Config struct {
	Project       string        `json:"project"`
	DynamicConfig DynamicConfig `json:"dynamic"`
	TraceLibrary  string        `json:"trace_lib"`
	ClearFolder   string        `json:"clear"`
	FetchDeps     bool          `json:"fetch_deps"`
}

type DynamicConfig struct {
	BenchmarkRegex        string     `json:"bench_regex"`
	WarmupIterations      int        `json:"wi"`
	MeasurementIterations int        `json:"i"`
	Duration              Duration   `json:"duration"`
	Runs                  int        `json:"runs"`
	Timeout               string     `json:"bench_timeout"`
	Regression            float32    `json:"regression"`
	Functions             []Function `json:"functions"`
	Rmit                  bool       `json:"rmit"`
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(data []byte) error {
	s := string(data)
	dur, err := time.ParseDuration(s[1 : len(s)-1])
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

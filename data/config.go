package data

import (
	"fmt"
	"time"
)

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
	BenchDuration         Duration   `json:"bench_duration"`
	BenchTimeout          string     `json:"bench_timeout"`
	BenchMem              bool       `json:"bench_mem"`
	Runs                  int        `json:"runs"`
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

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", time.Duration(d).String())), nil
}

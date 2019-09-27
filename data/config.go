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
	BenchTime             string     `json:"bench_time"`
	BenchTimeout          string     `json:"bench_timeout"`
	BenchDuration         Duration   `json:"bench_duration"`
	BenchMem              bool       `json:"bench_mem"`
	Runs                  int        `json:"runs"`
	RunDuration           Duration   `json:"run_duration"`
	Profile               Profile    `json:"profile"`
	ProfileDir            string     `json:"profile_dir"`
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

type Profile string

const (
	NoProfile   Profile = "none"
	AllProfiles Profile = "all"
	CPUProfile  Profile = "cpu"
	MemProfile  Profile = "mem"
)

var allProfiles = [...]string{string(NoProfile), string(AllProfiles), string(CPUProfile), string(MemProfile)}

func (p Profile) String() string {
	return string(p)
}

func (p *Profile) UnmarshalJSON(data []byte) error {
	s := string(data)
	s = s[1 : len(s)-1]

	if s == "" {
		*p = NoProfile
		return nil
	}

	for _, profile := range allProfiles {
		if s == profile {
			*p = Profile(s)
			return nil
		}
	}
	return fmt.Errorf("Invalid Profile '%s'", s)
}

func (p Profile) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", p)), nil
}

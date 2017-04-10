package data

type Config struct {
	Project       string        `json:"project"`
	DynamicConfig DynamicConfig `json:"dynamic"`
}

type DynamicConfig struct {
	WarmupIterations      int `json:"wi"`
	MeasurementIterations int `json:"i"`
	Runs                  int `json:"runs"`
}

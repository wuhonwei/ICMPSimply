package conf

type MeasureAgent struct {
	Settings *Host      `json:"settings" yaml:"settings"`
	Points   []*Machine `json:"points" yaml:"points"`
}

type Host struct {
	Hostname      string `json:"hostname" yaml:"hostname"`
	RTCPP         int    `json:"RTCPP" yaml:"RTCPP"`
	RUDPP         int    `json:"RUDPP" yaml:"RUDPP"`
	IsContinuity  bool   `json:"isContinuity" yaml:"isContinuity"`
	Mixture       bool   `json:"mixture" yaml:"mixture"`
	Step          int    `json:"step" yaml:"step"`
	DataSizeMB    int    `json:"dataSizeMB" yaml:"dataSizeMB"`
	MysqlAddress  string `json:"mysqlAddress" yaml:"mysqlAddress"`
	MysqlPassWord string `json:"mysqlPassWord" yaml:"mysqlPassWord"`
	UseDB         bool   `json:"useDB" yaml:"useDB"`
	MysqlUser     string `json:"mysqlUser" yaml:"mysqlUser"`
}

type Machine struct {
	Address             string `json:"address" yaml:"address"`
	Alias               string `json:"alias" yaml:"alias"`
	PeriodPacketNum     int    `json:"periodPacketNum" yaml:"periodPacketNum"`
	PerPacketIntervalMs int    `json:"perPacketIntervalMs" yaml:"perPacketIntervalMs"`
	PerPacketIntervalUs int    `json:"perPacketIntervalUs" yaml:"perPacketIntervalUs"`
	PeriodNum           int    `json:"periodNum" yaml:"periodNum"`
	PeriodMs            int    `json:"periodMs" yaml:"periodMs"`
	Size                int    `json:"size" yaml:"size"`
}

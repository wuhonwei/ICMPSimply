package conf

import "time"

//address.yml
type AddressConf struct {
	Rdb      *configRAMP  `yml:"rdb"`
	Ams      *configRAMP  `yml:"ams"`
	Job      *configJTIJ  `yml:"job"`
	Monapi   *configJTIJ  `yml:"monapi"`
	Transfer *configJTIJ  `yml:"transfer"`
	Tsdb     *configTsdb  `yml:"tsdb"`
	Index    *configJTIJ  `yml:"index"`
	Judge    *configJTIJ  `yml:"judge"`
	Prober   *configRAMP  `yml:"prober"`
	Agent    *configAgent `yml:"agent"`
}

//rdb ams monapi prober
type configRAMP struct {
	Http      string   `yml:"http"`
	Addresses []string `yml:"addresses"`
}

//job transfer index judge
type configJTIJ struct {
	Http      string   `yml:"http"`
	Rpc       string   `yml:"rpc"`
	Addresses []string `yml:"addresses"`
}

type configTsdb struct {
	Http string `yml:"http"`
	Rpc  string `yml:"rpc"`
}

//agent
type configAgent struct {
	Http string `yml:"http"`
}

//identity.yml
type IdentityConf struct {
	Ip    *configIdIp `yml:"ip"`
	Ident *configIdIp `yml:"ident"`
}

type configIdIp struct {
	Specify string `yml:"specify"`
	Shell   string `yml:"shell"`
}

//agent.yml
type AgentConf struct {
	Logger  *configLogger  `yml:"logger"`
	Sys     *configSys     `yml:"sys"`
	Enable  *configEnable  `yml:"enable"`
	Job     *configJob     `yml:"job"`
	Report  *configReport  `yml:"report"`
	Udp     *configUdp     `yml:"udp"`
	Metrics *configMetrics `yml:"metrics"`
}

type configLogger struct {
	Dir       string `yml:"dir"`
	Level     string `yml:"level"`
	KeepHours uint   `yml:"keepHours"`
}

type configEnable struct {
	Mon     bool `yml:"mon"`
	Job     bool `yml:"job"`
	Report  bool `yml:"report"`
	Metrics bool `yml:"metrics"`
}

type configUdp struct {
	Enable bool   `yml:"enable"`
	Listen string `yml:"listen"`
}

type configMetrics struct {
	MaxProcs         int  `yml:"maxProcs"`
	ReportIntervalMs int  `yml:"reportIntervalMs"`
	ReportTimeoutMs  int  `yml:"reportTimeoutMs"`
	ReportPacketSize int  `yml:"reportPacketSize"`
	SendToInfoFile   bool `yml:"sendToInfoFile"`
	Interval         time.Duration
}

type configJob struct {
	MetaDir  string `yaml:"metadir"`
	Interval int    `yaml:"interval"`
}

type configReport struct {
	Token    string            `yml:"token"`
	Interval int               `yml:"interval"`
	Cate     string            `yml:"cate"`
	UniqKey  string            `yml:"uniqkey"`
	SN       string            `yml:"sn"`
	Fields   map[string]string `yml:"fields"`
}

type configSys struct {
	Enable           bool                `yml:"enable"`
	IfacePrefix      []string            `yml:"ifacePrefix"`
	MountIgnore      *configMountIgnore  `yml:"mountIgnore"`
	IgnoreMetrics    []string            `yml:"ignoreMetrics"`
	IgnoreMetricsMap map[string]struct{} `yml:"-"`
	NtpServers       []string            `yml:"ntpServers"`
	Plugin           string              `yml:"plugin"`
	PluginRemote     bool                `yml:"pluginRemote"`
	Interval         int                 `yml:"interval"`
	Timeout          int                 `yml:"timeout"`
	FsRWEnable       bool                `yml:"fsRWEnable"`
}

type configMountIgnore struct {
	Prefix  []string `yml:"prefix"`
	Exclude []string `yml:"exclude"`
}

package config

import (
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"ICMPSimply_bingfa/ICMPSimply/redis"
	"ICMPSimply_bingfa/ICMPSimply/state"
	"bufio"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

var logger = mylog.GetLogger()

const configSize = 1000 //大小定为1000
const protocol = "icmp"

type Config struct {
	Data   Data   `json:"settings" yaml:"settings"` //数据库保存
	Points []King `json:"points" yaml:"points"`
}
type King struct {
	Id     int     `json:"slot" yaml:"slot"`
	Points []Point `json:"send_task" yaml:"send_task"`
}

func (c Config) Len() int {
	return len(c.Points)
}
func (c Config) Swap(i, j int) {
	c.Points[i], c.Points[j] = c.Points[j], c.Points[i]
}
func (c Config) Less(i, j int) bool {
	return c.Points[i].Id < c.Points[j].Id
}

type Data struct {
	Hostname      string `json:"hostname" yaml:"hostname"`
	RTCPP         int    `json:"RTCPP" yaml:"RTCPP"`
	RUDPP         int    `json:"RUDPP" yaml:"RUDPP"`
	IsContinuity  bool   `json:"isContinuity" yaml:"isContinuity"`
	Mixture       bool   `json:"mixture" yaml:"mixture"`
	MyPublicIp    string
	Step          int64  `json:"step" yaml:"step"`
	DataSizeMB    int64  `json:"dataSizeMB" yaml:"dataSizeMB"`
	MysqlAddress  string `json:"mysqlAddress" yaml:"mysqlAddress"`
	MysqlUser     string `json:"mysqlUser" yaml:"mysqlUser"`
	MysqlPassWord string `json:"mysqlPassWord" yaml:"mysqlPassword"`
	UseDB         bool   `json:"useDB" yaml:"useDB"`
	//DBName        string `json:"dbName" yaml:"dbName"`
}

//每个机器的信息
type Point struct {
	Address string `json:"address" yaml:"address"`
	Alias   string `json:"alias" yaml:"alias"`
	Type    string //`json:"type" yaml:"type"`
	Size    int    `json:"size"` //不读取yml的size
	//每个周期包数
	PeriodPacketNum     uint64 `json:"periodPacketNum" yaml:"periodPacketNum"`
	PerPacketIntervalMs int64  `json:"perPacketIntervalMs" yaml:"perPacketIntervalMs"`
	PerPacketIntervalUs int64  `json:"perPacketIntervalUs" yaml:"perPacketIntervalUs"`
	//间隔一般为0
	PeriodMs uint64 `json:"periodMs" yaml:"periodMs"`
	//发多少周期默认以前
	PeriodNum int64 `json:"periodNum" yaml:"periodNum"`
}
type cfgJson struct {
	Hostname string `json:"hostname" yaml:"alias"`
	Ip       string `json:"ip" yaml:"address"`
	//MeasureAgent string `yaml:"measureAgent"`
}

// LoadConfig 完成对外界配置的读取与检测，完成功能如下：
// (1) 从指定路径的配置文件获取配置信息 file 为测量配置，cfsFilename 为cfg文件路径
// (2) 删除自己的节点
// (3) 对每个需要测量的IP地址进行连通性测试
// (4) 检测设定的测量步长step是否满足要求
// (5) 删除同一数据中心的其他节点
func LoadConfig(configurationFilename string, cfsFilename string) (*Config, error) {
	//conf := Config{}
	conf, err := GetMeasureAgJson(configurationFilename, cfsFilename)
	// 不配置 size或者size=0,同时把协议固定
	generate64byteAnd1024Byte(conf)
	//删除重复的和自己测量自己的
	deleteMySelfPoint(conf)
	//检测ip是否正常
	IPCheck(conf)
	/*
		以下有个函数因为任务编排修改了配置文件被暂时注释
	*/
	//检测保存时间是否大于测量时间
	//checkStep(conf)
	return conf, err
}

//func checkStep(conf *Config) {
//	if conf.Data.Step == -1 {
//		return
//	}
//	var timeMillion int64
//	step := conf.Data.Step
//	for _, point := range conf.Points {
//		//所有发包数乘以发包时间，毫秒数
//		timeMillion += int64(point.PeriodPacketNum) * (point.PerPacketIntervalMs)
//	}
//	if conf.Points[0].Size == 0 {
//		//Duration是纳秒，但是乘以time.Second单位变为秒
//		if time.Duration(step)*time.Second <= (time.Duration(timeMillion)*time.Millisecond+2*time.Second)*2 {
//			//logger.Error("step must more than measure time,step= more than,measure time",
//			//	zap.Int64("step", step),
//			//	zap.Any("time.Duration(timeMillion)*time.Millisecond/time.Second+2) ", time.Duration(2*timeMillion/1e3)*time.Second))
//			measureTime := time.Duration(2*timeMillion/1e3) * time.Second
//			logger.Error(fmt.Sprintf("manage\tstep must more than measure time. step:%v,measure time:%v.\tcpu:%v,mem:%v", step, measureTime, state.LogCPU, state.LogMEM))
//
//			os.Exit(1)
//		}
//	}
//}

func IPCheck(config *Config) {
	// 域名或IP检测
	for _, king := range config.Points {
		for _, point := range king.Points {
			if _, err := net.LookupHost(point.Address); err != nil {
				//logger.Warn("point can't access, please check your configuration", zap.Any("point", point))
				logger.Warn(fmt.Sprintf("manage\tpoint can't access, please check your configuration\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
			}
		}
	}
}

func deleteMySelfPoint(conf *Config) {

	for i, king := range conf.Points {
		points := make([]Point, 0)
		for _, point := range king.Points {
			s := strings.Split(point.Alias, "_")
			str := ""
			if len(s) > 2 {
				str = s[0] + "_" + s[1]
			}
			//两个地址不相同
			if !strings.EqualFold(point.Address, conf.Data.MyPublicIp) {
				//再判断主机名是否相同
				if str == "" || !strings.Contains(conf.Data.Hostname, str) {
					points = append(points, point)
				}
			}
		}
		conf.Points[i].Points = points
	}

}

func scanner() (s string) {
	newScanner := bufio.NewScanner(os.Stdin)
	for newScanner.Scan() {
		s = newScanner.Text()
		return
	}
	return
}
func GetMeasureAgJson(configurationFilename, cfsFilename string) (*Config, error) {
	conf := &Config{}
	conf.Data.Hostname, conf.Data.MyPublicIp = GetCfgJson(cfsFilename)
	//configBytes, err :=redis.Value("106.3.133.6:6790","Ali_Dubai")
	configBytes, err := redis.Value("106.3.133.6:6790", conf.Data.Hostname)
	if configBytes == nil {
		configBytes, err = ioutil.ReadFile(configurationFilename)
		if err != nil {
			return nil, err
		}
	}
	//configBytes, err = ioutil.ReadFile(configurationFilename)
	//var stdin string
	//_,err = fmt.Scan(&stdin)
	//fmt.Println(stdin)
	//configBytes, _ = base64.StdEncoding.DecodeString(stdin)
	//fmt.Println(encoded)
	//fmt.Println(string(byteArray))
	//fmt.Println()
	//err = json.Unmarshal(configBytes, conf)
	err = yaml.Unmarshal(configBytes, conf)
	if err != nil {
		//logger.Error("can't find conf configurationFilename ", zap.String("filename", configurationFilename))
		logger.Error(fmt.Sprintf("manage\tcan't find conf configurationFilename\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	}
	//解密sl
	//conf.Data.MysqlPassWord="203club."
	conf.Data.MysqlPassWord = AesDecrypt(conf.Data.MysqlPassWord, Key)
	conf.Data.MysqlAddress = AesDecrypt(conf.Data.MysqlAddress, Key)
	return conf, err
}
func GetCfgJson(cfgFilename string) (string, string) {
	cfg := &cfgJson{}
	hostnameBytes, err := ioutil.ReadFile(cfgFilename)
	//err = json.Unmarshal(hostnameBytes, cfg)
	err = yaml.Unmarshal(hostnameBytes, cfg)
	if err != nil {
		//logger.Error("get hostname from cfg.json fail:%v", zap.Error(err))
		logger.Error(fmt.Sprintf("manage\tfail to get hostname from cfg.yml\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	}
	return cfg.Hostname, cfg.Ip
}

//因为任务编排被注释的老函数，下面有个新的
//func generate64byteAnd1024Byte(conf *Config) {
//	points := make([]Point, 0)
//	for _, point := range conf.Points {
//		point.Type = protocol
//		if point.Size == 0 {
//			point.Size = 64
//			points = append(points, point)
//			point.Size = 1024
//			point.PeriodPacketNum = 50
//			point.PerPacketIntervalMs = 20
//			points = append(points, point)
//		} else {
//			points = append(points, point)
//		}
//	}
//	conf.Points = points
//}

func generate64byteAnd1024Byte(conf *Config) {
	//fmt.Println(conf.Points[0].Points[0].Type)
	for i, ki := range conf.Points {
		points := make([]Point, 0)
		for _, point := range ki.Points {
			point.Type = protocol
			if point.Size == 0 {
				point.Size = 64
				points = append(points, point)
				point.Size = 1024
				point.PeriodPacketNum = 50
				point.PerPacketIntervalMs = 20
				points = append(points, point)
			} else {
				points = append(points, point)
			}
		}
		conf.Points[i].Points = points
	}
}

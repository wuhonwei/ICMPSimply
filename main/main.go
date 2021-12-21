package main

import (
	"ICMPSimply_bingfa/ICMPSimply/config"
	"ICMPSimply_bingfa/ICMPSimply/listening"
	"ICMPSimply_bingfa/ICMPSimply/measure"
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"ICMPSimply_bingfa/ICMPSimply/state"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sort"
)

const (
	winDirOfConfig   = "./config/Ali_bombay-config.yaml"
	winDirOfCFG      = "./config/cfg.yaml"
	linuxDirOfConfig = "/home/n9e/plugin/RenWu.yml"
	linuxDirOfCFG    = "/root/point.yml"
)

var (
	configurationFilename string
	win10                 = runtime.GOOS == "windows"
	logger                = mylog.GetLogger()
)

func init() {
	if win10 {
		flag.StringVar(&configurationFilename, "config", winDirOfConfig, "config is invalid, use default.json replace")
	} else {
		flag.StringVar(&configurationFilename, "config", linuxDirOfConfig, "config is invalid, use default.json replace")
	}
	//flag.Parse()
}

func main() {
	go state.CheckCPUAndMem()

	f, err := os.Create("cpu.prof")
	if err != nil {

	}

	// 获取系统信息
	if err := pprof.StartCPUProfile(f); err != nil { //监控cpu

	}
	defer pprof.StopCPUProfile()

	//go func() {
	//	//logger.Info("%v", zap.Error(http.ListenAndServe("0.0.0.0:7070", nil)))
	//	logger.Info("%v", zap.Error(http.ListenAndServe("0.0.0.0:7070", nil)))
	//}()
	//flag.Parse()
	var cfgFilename string
	if win10 {
		cfgFilename = "./config/cfg.yaml"
	} else {
		cfgFilename = linuxDirOfCFG
	}
	conf, err := config.LoadConfig(configurationFilename, cfgFilename)
	if err != nil {
		//log.Fatalf("manage\tfail to load config\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM)
		return
	}
	sort.Sort(conf)
	if !win10 {
		listening.ServerListen(conf) //开启服务器监听
	}
	confChan := make(chan config.Config, 10)                                 //带缓存的channel，无缓存的channel的读写都将进行堵塞
	config.DynamicUpdateConfig(configurationFilename, cfgFilename, confChan) //Linux赋权限和更新配置

	//logger.Debug("init end, wait server starting ...", zap.String("time", time.Since(start).String()))
	logger.Debug(fmt.Sprintf("manage\tinit end, wait to server starting ...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	//listening.WaitServer(&conf) //等待其他节点启动
	//logger.Debug("wait server end,start measuring ...", zap.String("time", time.Since(start).String()))
	logger.Debug(fmt.Sprintf("manage\tstart measuring ...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	measure.Measure(conf, confChan)
	//logger.Info("end measuring ...", zap.String("time", time.Since(start).String()))
	logger.Info(fmt.Sprintf("manage\tend measuring...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))

	f1, err := os.Create("mem.prof")
	if err != nil {

	}
	//runtime.GC()                                       // GC，获取最新的数据信息
	if err := pprof.WriteHeapProfile(f1); err != nil { // 写入内存信息

	}
	f1.Close()
	f2, err := os.Create("goroutine.prof")
	if err != nil {

	}
	if gProf := pprof.Lookup("goroutine"); gProf == nil {
	} else {
		gProf.WriteTo(f2, 0)
	}
	f2.Close()
}

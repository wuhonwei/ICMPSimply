package measure

import (
	"ICMPSimply_bingfa/ICMPSimply/config"
	"ICMPSimply_bingfa/ICMPSimply/dataStore/dao"
	"ICMPSimply_bingfa/ICMPSimply/etcd"
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"ICMPSimply_bingfa/ICMPSimply/protocol"
	"ICMPSimply_bingfa/ICMPSimply/state"
	"ICMPSimply_bingfa/ICMPSimply/statisticsAnalyse"
	"encoding/json"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var logger = mylog.GetLogger()
var wg sync.WaitGroup
var lock sync.Mutex
var etcdLock sync.Mutex
var failNumLock sync.Mutex
var cells []statisticsAnalyse.Cell
var failNumInSerier float64
var failNumInPeriodPacketNum float64
var totalNum float64

func Measure(conf *config.Config, confChan chan config.Config) {
	//message := make([]protocol.Message, len(conf.Points)) //用来保存每个节点的测量配置
	//cells := make([]statisticsAnalyse.Cell, 0)            //用来保护统计数据
	//var timePushStart time.Time

	res := returnRes()
	//res := make([]float64, 15)
	// 测量失败将值置为-9
	//for i := range res {
	//	res[i] = -9
	//}
	curTime := time.Now().Unix()
	// 顺序打乱 降低流量汇集概率，
	/*zm
	seed 只用于决定一个确定的随机序列。不管seed多大多小，只要随机序列一确定，本身就不会再重复。除非是样本空间太小。解决方案有两种：

	在全局初始化调用一次seed即可
	每次使用纳秒级别的种子（强烈不推荐这种）在高并发下，即使使用UnixNano作为解决方案，同样会得到相同的时间戳，Go官方也不建议在服务中同时调用。
	crypto/rand是为了提供更好的随机性满足密码对随机数的要求，
	在linux上已经有一个实现就是/dev/urandom，
	crypto/rand 就是从这个地方读“真随机”数字返回，但性能比较慢,相差10倍左右
	*/
	rand.Seed(time.Now().UnixNano())
	//if conf.Data.Mixture {
	//	lock.Lock()
	//	// 洗牌算法，生成随机不重复的序列
	//	perm := rand.Perm(len(conf.Points))
	//	points := make([]config.Point, len(conf.Points)) //临时变量
	//	for i, p := range perm {
	//		points[i] = conf.Points[p]
	//	}
	//	conf.Points = points //把顺序写回
	//	lock.Unlock()
	//}
	cellsPush := make([]statisticsAnalyse.Cell, 0)

	cli, err := etcd.NewCltConn(etcd.NewLeaseInfo(60))
	if err != nil {
		logger.Error(fmt.Sprintf("etcd get cli fail,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
	}
	etcdLock.Lock()
	_, err = cli.PutService("/measure/slot", strconv.Itoa(len(conf.Points)))
	_, err = cli.PutService("/measure/result/"+conf.Data.Hostname, "100")
	etcdLock.Unlock()
	if err != nil {
		logger.Error(fmt.Sprintf("etcd PutService fail,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
	}

	watchKey := cli.Client.Watch(cli.Client.Ctx(), "/measure/"+conf.Data.Hostname, clientv3.WithPrefix())
	for recv := range watchKey {
		logger.Info(string(recv.Events[0].Kv.Key) + fmt.Sprintf("etcd change,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
		_, err = cli.PutService("/measure/result/"+conf.Data.Hostname, fmt.Sprintf("%f", 0.0))
		Serier, err := strconv.Atoi(string(recv.Events[0].Kv.Value))
		if err != nil {
			logger.Error(fmt.Sprintf("etcd get Serier fail,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
		}
		if Serier == -1 || Serier >= len(conf.Points) {
			err = dao.DeadDB()
			if err != nil {

			}
			for i, _ := range cellsPush {
				//返回子串str在字符串s中第一次出现的位置。
				//如果找不到则返回-1；如果str为空，则返回0
				index := strings.Index(cellsPush[i].Tags, ",p_seq=")
				cellsPush[i].Tags = cellsPush[i].Tags[:index]
			}
			pushToDB(cellsPush, conf)
			pushData, err := json.Marshal(cellsPush)
			if err != nil {
				logger.Error(fmt.Sprintf("wrong in transfer to json,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
			}
			fmt.Println(string(pushData))
			return
		} else if Serier >= 0 && Serier < len(conf.Points) {
			cells = make([]statisticsAnalyse.Cell, 0)
			i := Serier
			wg.Add(len(conf.Points[i].Points))
			message := make([]protocol.Message, len(conf.Points[i].Points))

			failNumInSerier, totalNum = 0, float64(len(conf.Points[i].Points))
			curTime = time.Now().Unix()
			for j := 0; j < len(conf.Points[i].Points); j++ {
				j := j
				go Gao(message, j, i, curTime, res, conf, cli)
			}
			wg.Wait()
			etcdLock.Lock()
			if totalNum == 0 {
				_, err = cli.PutService("/measure/result/"+conf.Data.Hostname, fmt.Sprintf("%f", 1.0))
			} else {
				_, err = cli.PutService("/measure/result/"+conf.Data.Hostname, fmt.Sprintf("%f", (totalNum-failNumInSerier)/totalNum))
			}
			etcdLock.Unlock()
			if err != nil {
				logger.Error(fmt.Sprintf("etcd PutService fail,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
			}
			for j := range cells {
				cells[j].Timestamp = curTime
			}
			cellsPush = append(cellsPush, cells...)
			cells = nil
		} else {
			logger.Error(fmt.Sprintf("No such slot,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
		}
	}
	//for i := 0; i < len(conf.Points); i++ {
	//	wg.Add(len(conf.Points[i].Points))
	//	message := make([]protocol.Message, len(conf.Points[i].Points))
	//	cells = make([]statisticsAnalyse.Cell, 0)
	//	i:=i
	//	failNum,totalNum=0, float64(len(conf.Points[i].Points))
	//	for j:=0;j< len(conf.Points[i].Points);j++{
	//		j := j
	//		go Gao(message,j,i,curTime,res,conf,cli)
	//	}
	//	wg.Wait()
	//	etcdLock.Lock()
	//	_, err = cli.PutService("/measure/result/"+conf.Data.Hostname, strconv.FormatFloat(failNum/totalNum, 'E', -1, 64))
	//	etcdLock.Unlock()
	//	if err!=nil{
	//		logger.Error(fmt.Sprintf("etcd PutService fail,hostname %v\tcpu:%v,mem:%v", conf.Data.Hostname, state.LogCPU, state.LogMEM))
	//	}
	//
	//	for j := range cells {
	//		cells[j].Timestamp = curTime
	//	}
	//	pushToDB(cells, conf)
	//	cellsPush=append(cellsPush,cells...)
	//	cells=nil
	//}
	//err = dao.DeadDB()
	//if err != nil {
	//
	//}
	////pushToDB(cells, conf)
	////storeToFile.SavePerPacketToLocal(recv, conf, i) // 将得数据写入文件
	////fmt.Println(len(cellsPush))
	//for i, _ := range cellsPush {
	//	//返回子串str在字符串s中第一次出现的位置。
	//	//如果找不到则返回-1；如果str为空，则返回0
	//	index := strings.Index(cellsPush[i].Tags, ",p_seq=")
	//	cellsPush[i].Tags = cellsPush[i].Tags[:index]
	//}
	//pushData, err := json.Marshal(cellsPush)
	////if err != nil {
	////	//logger.Error("wrong in transfer to json", zap.Error(err))
	////	//logger.Error("wrong in transfer to json", zap.Error(err))
	////}
	//fmt.Println(string(pushData))
	////cells = nil
}

func Gao(message []protocol.Message, j, i int, curTime int64, res []float64, conf *config.Config, cli *etcd.CltConn) {
	message[j].Address = conf.Points[i].Points[j].Address
	message[j].Endpoint = conf.Data.Hostname
	message[j].Size = conf.Points[i].Points[j].Size
	message[j].Sequence = 1
	message[j].Protocol = conf.Points[i].Points[j].Type
	message[j].Step = conf.Data.Step
	message[j].PeriodPacketNum = conf.Points[i].Points[j].PeriodPacketNum
	message[j].Alias = conf.Points[i].Points[j].Alias
	message[j].PercentPacketIntervalMs = conf.Points[i].Points[j].PerPacketIntervalMs // 包间隔 发送超时设定
	message[j].PercentPacketIntervalUs = conf.Points[i].Points[j].PerPacketIntervalUs // 包间隔 发送超时设定
	message[j].Hostname = conf.Data.Hostname
	message[j].PeriodMs = conf.Points[i].Points[j].PeriodMs
	message[j].PeriodNum = conf.Points[i].Points[j].PeriodNum
	message[j].PacketSeq = i

	var measureFun func(*protocol.Message, net.Conn, *sync.Map)
	//var wg sync.WaitGroup
	var conn net.Conn
	var err error
	conn, err = net.DialTimeout("ip4:icmp", message[j].Address, time.Second) //超时时间为1s
	// icmp测量失败 后序该点测量直接略过 设为-9
	if err != nil {
		//logger.Error(conf.Data.Hostname+"_to_"+message[i].Alias+" network connect fail:", zap.Error(err))
		logger.Error(fmt.Sprintf("send\tnetwork connect fail,hostname %v to %v\tcpu:%v,mem:%v", conf.Data.Hostname, message[j].Alias, state.LogCPU, state.LogMEM))
		lock.Lock()
		cells = append(cells, generateFailMeasureCells(&message[j], res)...)
		lock.Unlock()
		failNumLock.Lock()
		failNumInSerier++
		failNumLock.Unlock()
		//待修改
		return
	}
	measureFun = protocol.SendICMPPacket
	// 发包 如果size配为0则发送一次64再发送一次1024
	var recv *sync.Map
	for k := int64(0); k < message[j].PeriodNum; k++ { //每个点->每个协议->每个周期
		//一般PeriodNum都为1，所以其实只发了一次
		recv = nil
		recv = new(sync.Map)
		//startTest := time.Now()
		//待修改
		//logger.Debug("init message info end, start sending ...", zap.String("time", time.Since(startTest).String()))
		logger.Debug(fmt.Sprintf("send\tinit message info end, start sending ...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		//真正发包
		cells1, _, _ := task(measureFun, &message[j], conn, recv) //分别传入测量函数、测量节点配置、链接、包序号map
		//logger.Debug("end sending, ...", zap.String("init", time.Since(startTest).String()))
		logger.Debug(fmt.Sprintf("send\tend sending ...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		//每次测量需要的暂停时间，即发包间隔
		time.Sleep(time.Duration(message[j].PeriodMs) * time.Millisecond)
		// 上传到openfalcon的统计值
		for n := range cells1 {
			cells1[n].SourceIp = conf.Data.MyPublicIp
			cells1[n].DestIp = message[j].Address
		}
		lock.Lock()
		cells = append(cells, cells1...)
		lock.Unlock()
		if cells1 == nil || cells1[0].Value == -9 {
			logger.Error(fmt.Sprintf("send\tmeasure fail. hostname:%v,addr:%v\tcpu:%v,mem:%v", message[j].Alias, message[j].Protocol, state.LogCPU, state.LogMEM))
			continue
		} else if message[j].Size == 1024 {
			logger.Debug(fmt.Sprintf("send\tmeasure success. hostname:%v,addr:%v\tcpu:%v,mem:%v", message[j].Alias, message[j].Protocol, state.LogCPU, state.LogMEM))
		}
		//logger.Debug("runtime.NumGoroutine()", zap.Int("num", runtime.NumGoroutine()))
		//logger.Info("发送情况", zap.String("hostname", message[i].Alias), zap.Int("size", message[i].Size), zap.Int("j", j))
		//if runtime.GOOS == "windows" {
		//logger.Debug("start to perData save to  file ...", zap.String("time", time.Since(startTest).String()))
		logger.Debug(fmt.Sprintf("store\tstart to save perData to file\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		// 存储每个探测包到本地
		//storeToFile.SavePerPacketToLocal(recv, conf, i) // 将得数据写入文件
		logger.Debug(fmt.Sprintf("store\tend save\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		recv = nil
		cells1 = nil
		message[j].Sequence = 1
	}
	//if i == len(message)-1 { //是否开启循环测量的处理，相等时为测完所有节点
	//	//startTest := time.Now()
	//	cells = deleteNull(cells) // 去除空值
	//	//logger.Debug("marshal result 	starting ...", zap.String("time", time.Since(startTest).String()))
	//	logger.Debug(fmt.Sprintf("receive\tmarshal result starting ...\tstarting ...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	//	for i := range cells {
	//		cells[i].Timestamp = curTime
	//	}
	//	//logger.Debug("end marshal ...", zap.String("time", time.Since(startTest).String()))
	//	logger.Debug(fmt.Sprintf("receive\tend marshal ...\tstarting ...\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	//	//传入数据库
	//	//pushToDB(cells, conf)
	//	//for i, _ := range cells {
	//	//	//返回子串str在字符串s中第一次出现的位置。
	//	//	//如果找不到则返回-1；如果str为空，则返回0
	//	//	index := strings.Index(cells[i].Tags, ",p_seq=")
	//	//	cells[i].Tags = cells[i].Tags[:index]
	//	//}
	//	//pushData, err := json.Marshal(cells)
	//	//if err != nil {
	//	//	logger.Error("wrong in transfer to json", zap.Error(err))
	//	//}
	//	//cells = nil
	//	lenConfChan := len(confChan)
	//	// 更新配置
	//	if len(confChan) != 0 {
	//		for l := 0; l < lenConfChan-1; l++ {
	//			<-confChan
	//		}
	//		newConf := <-confChan
	//		lock.Lock()
	//		*conf = newConf
	//		lock.Unlock()
	//		//logger.Info("update config ", zap.Any("conf", conf))
	//		logger.Info(fmt.Sprintf("manage\tupdate config\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	//	}
	//	// 连续模式
	//	if conf.Data.IsContinuity {
	//		////pushToFalcon(pushData) //直接push到服务器
	//		//timeStepSecond := time.Duration(conf.Data.Step-2)*time.Second - time.Since(timePushStart)
	//		//logger.Debug("timeStepSecond ", zap.Any("timeStepSecond", timeStepSecond))
	//		//i = -1
	//		//// 严格保证step
	//		//if timeStepSecond > 0 {
	//		//	//time.Sleep(timeStepSecond)
	//		//}
	//		////getTCPConn(conf, tcpConn)
	//	} else {
	//		// 如果模式改变，需要改变监听方式 根据监听的端口是否被占用作为标准，即使模式切换仍能保证端口有能力回显数据
	//		//logger.Debug("output result to terminal starting ...", zap.String("time", time.Since(startTest).String()))
	//		//logger.Debug("output result to terminal starting ...", zap.String("time", time.Since(startTest).String()))
	//		listening.ServerListen(conf)
	//		// 打印到终端，openfalcon自己收集
	//		//str := string(pushData)
	//		//fmt.Println(str)
	//		//fmt.Println(string(pushData))
	//		//logger.Debug("end output ...", zap.String("time", time.Since(startTest).String()))
	//	}
	//	//wg.Wait()
	//}
	wg.Done()
}

func deleteNull(cells []statisticsAnalyse.Cell) []statisticsAnalyse.Cell {
	counter := 0
	for _, cell := range cells {
		if !strings.EqualFold("", cell.Tags) {
			counter++
		}
	}
	cellsCopy := make([]statisticsAnalyse.Cell, counter)
	for i, cell := range cells {
		if !strings.EqualFold("", cell.Tags) {
			cellsCopy[i] = cell
		}
	}
	return cellsCopy
}
func returnRes() []float64 {
	res := make([]float64, 15)
	for i := range res {
		res[i] = -9
	}
	return res
}
func pushToDB(cells []statisticsAnalyse.Cell, conf *config.Config) {
	dao.Store(cells, conf)
}

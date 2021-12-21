package protocol

import (
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"ICMPSimply_bingfa/ICMPSimply/state"
	"ICMPSimply_bingfa/ICMPSimply/statisticsAnalyse"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	EchoReplyIPHeaderLength = 20
	EchoReplyType           = 0
	EchoRequestType         = 8
	EchoSendFastType        = 4
)

var logger = mylog.GetLogger()

func RecvICMPPacket(message *Message, conn net.Conn, recv *sync.Map) {
	//defer logger.Debug("goroutine return:", zap.String("message.Protocol", message.Protocol), zap.String("message.Address", message.Address))
	defer logger.Debug(fmt.Sprintf("receive\tgoroutine return. address:%v\tcpu:%v,mem:%v", message.Address, state.LogCPU, state.LogMEM))
	id0, id1 := genIdentifier(os.Getpid() & 0xffff) //父进程id作为唯一标识
	//var id0, id1 byte = 11, 11
	//var timeout int64 = 2000
	//在使用Go语言的net.Dial函数时，发送echo request报文时，不用考虑i前20个字节的ip头；
	// 但是在接收到echo response消息时，前20字节是ip头。后面的内容才是icmp的内容，应该与echo request的内容一致
	var receive []byte
	receive = make([]byte, message.Size+20)
	//reader := bufio.NewReader(conn)
	// 如果由于调度问题导致协程积累，定时清除
	start := time.Now()
	timeout := time.Duration(message.PercentPacketIntervalMs*int64(message.PeriodPacketNum))/1000*time.Second + time.Second*2
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	for !message.IsFinish && time.Since(start) < timeout {
		var endDuration int64 = 5
		//n, err := reader.Read(receive)
		// 表现为阻塞，实际上是非阻塞
		n, err := conn.Read(receive)

		if err != nil || n < 0 {
			if err != nil && strings.Contains(err.Error(), "timeout") {
				return
			}
			//logger.Error("icmp read error", zap.Error(err))
			logger.Error(fmt.Sprintf("receive\ticmp read error\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
			continue
		}
		receive = receive[:n]
		//zm检验icmp是不是自己的
		if id0 == receive[EchoReplyIPHeaderLength+4] &&
			id1 == receive[EchoReplyIPHeaderLength+5] {
			seq := getSeq(receive[EchoReplyIPHeaderLength+6], receive[EchoReplyIPHeaderLength+7]) // 提取数据包编号
			//logger.Infof("%v hope remote addr = %v,recv seq = %d", time.Now(), message.Address, seq)
			seqRecv, ok := recv.Load(seq)
			if !ok {
				//logger.Error("recv[index] is nil", zap.String("message.Address", message.Address), zap.Uint64("seq", seq))
				logger.Error(fmt.Sprintf("receive\trecv[index] is nil.address:%v,seq:%v\tcpu:%v,mem:%v", message.Address, seq, state.LogCPU, state.LogMEM))
			}
			seqRes, ok := seqRecv.(statisticsAnalyse.RecvStatic)
			if !ok {
				//logger.Error("seqRes's type isn't RecvStatic")
				logger.Error(fmt.Sprintf("receive\tseqRes's type isn't RecvStatic\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
			}
			//除了判断err!=nil，还有判断请求和应答的ID标识符，sequence序列码是否一致，以及ICMP是否超时（receive[ECHO_REPLY_HEAD_LEN] == 11，即ICMP报头的类型为11时表示ICMP超时）
			var ttl int
			//??
			ttl = int(receive[8])
			// 解析收到的正确icmp包,需要去除重复的包
			if !seqRes.IsValid {
				endTime := time.Now()
				res := statisticsAnalyse.RecvStatic{
					Seq:           seq,
					Alias:         message.Alias,
					Proto:         message.Protocol,
					Size:          message.Size,
					RTT:           -1,
					TTL:           ttl,
					SendTimeStamp: seqRes.SendTimeStamp,
					RecvTimeStamp: endTime.UnixNano() / 1e3, // us
					IsValid:       true,
				}
				endDuration = res.RecvTimeStamp - res.SendTimeStamp
				if ttl == -1 {
					res.IsValid = false
				}
				res.RTT = float64(endDuration) / 1e3
				//logger.Infof("%v,seq = %v %v RTT = %v", message.Alias, seq, message.Protocol, res.RTT)
				recv.Store(seq, res)
				sendFeedback(message, message.Protocol, endDuration, ttl, fmt.Sprint(seq))
			}
		}
	}
	err := conn.Close()
	if err != nil {
		//logger.Error(" connection close fail:", zap.String("Address", message.Address), zap.String("Protocol", message.Protocol), zap.Error(err))
		logger.Error(fmt.Sprintf("manage\tconnection close fail.Address:%v\tcpu:%v,mem:%v", message.Address, state.LogCPU, state.LogMEM))
	}
}
func SendICMPPacket(message *Message, conn net.Conn, recv *sync.Map) {
	if recv == nil {
		//logger.Error("recv is empty:", zap.Any("recv", recv))
		logger.Error(fmt.Sprintf("send\trecv is empty\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	}
	icmp(message, conn, recv)
	message.Sequence += 1
	if message.Sequence > message.PeriodPacketNum {
		message.Sequence = 1 //发送完一个周期的探测包过后重新置为1，为下次发包做准备
	}
}
func icmp(message *Message, conn net.Conn, recv *sync.Map) {
	//const EchoReplyIPHeaderLength = 20
	id0, id1 := genIdentifier(os.Getpid() & 0xffff)
	//var id0, id1 byte = 11, 11
	msg := make([]byte, message.Size)
	fillLength := message.Size - EchoRequestType - EchoReplyIPHeaderLength
	//	logger.Debugf("ppid = %v", os.Getpid()&0xfff)
	msg[0] = 8                                      // echo,第一个字节表示报文类型，8表示回显请求,0表示回显的应答
	msg[1] = 0                                      // code 0,ping的请求和应答，该code都为0
	msg[2] = 0                                      // checksum
	msg[3] = 0                                      // checksum
	msg[4], msg[5] = id0, id1                       //identifier[0] identifier[1], ID标识符 占2字节
	msg[6], msg[7] = genSequence(&message.Sequence) //sequence[0], sequence[1],序号占2字节
	length := message.Size
	fillRandData(fillLength, msg)
	check := checkSum(msg[0:length]) //计算检验和。
	msg[2] = byte(check >> 8)
	msg[3] = byte(check & 255)

	//conn, err := net.Dial("ip4:icmp", message.Address)
	//conn, err := net.DialTimeout("ip:icmp", message.Address, time.Duration(timeout*1000*1000))
	//	logger.Debugf("addr = %v,recv seq = %d", message.Address, message.Sequence)
	startTime := time.Now()
	res := statisticsAnalyse.RecvStatic{
		Seq:           message.Sequence,
		Alias:         message.Alias,
		Proto:         message.Protocol,
		Size:          message.Size,
		RTT:           -1,
		TTL:           0,
		SendTimeStamp: startTime.UnixNano() / 1e3, //转化到us ,运行时这个也是第一个运行先取得这个函数的返回值
		RecvTimeStamp: 0,
		IsValid:       false,
	}
	n, err := conn.Write(msg[0:length]) //onn.Write方法执行之后也就发送了一条ICMP请求，同时进行计时和计次
	recv.Store(message.Sequence, res)   //非值传递，value为interface{}空接口，包含类型和类型的值(指针)两个字节，参见gopl P243
	if err != nil || n <= 0 {
		//logger.Error(" network send fail:", zap.Uint64("message.Sequence", message.Sequence), zap.Error(err))
		logger.Error(fmt.Sprintf("send\tnetwork send fail.Sequence:%v\tcpu:%v,mem:%v", message.Sequence, state.LogCPU, state.LogMEM))
		return
	}
	//	logger.Infof("send time = %v", startTime.UnixNano()/1e6)
}
func genIdentifier(id int) (byte, byte) {
	return uint8(id >> 8), uint8(id & 0xff)
}

func genSequence(v *uint64) (byte, byte) {
	ret := make([]byte, 8)
	if *v > 65000 {
		//logger.Warn("seq overflow 65000, but it only effect icmp", zap.Uint64("seq", *v))
		logger.Warn(fmt.Sprintf("send\tseq overflow 65000, but it only effect icmp\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		return 0, 0
	}
	binary.LittleEndian.PutUint64(ret, *v)
	return ret[1], ret[0]
}
func fillRandData(fillLength int, msg []byte) {
	b := make([]byte, fillLength)
	rand.Read(b)
	for i := EchoReplyIPHeaderLength + EchoRequestType; i < len(msg); i++ {
		msg[i] = b[i-EchoRequestType-EchoReplyIPHeaderLength]
	}
}
func checkSum(data []byte) uint16 {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += sum >> 16

	return uint16(^sum)
}
func getSeq(seq1, seq2 byte) uint64 {
	seq := make([]byte, 8)
	seq[1] = seq1
	seq[0] = seq2
	seqUint64 := binary.LittleEndian.Uint64(seq)
	return seqUint64
}

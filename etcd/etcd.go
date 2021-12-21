package etcd

import (
	"ICMPSimply_bingfa/ICMPSimply/conf"
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.uber.org/zap"
	"sync"
	"time"
)

type CltConn struct {
	Client *clientv3.Client
	lease  *LeaseInfo
	lock   sync.Mutex
}

type LeaseInfo struct {
	//租约什么时候被释放,单位为second
	aliveTime int64
	//这个心跳持续多久， -1就是永远都不放弃，直到main执行完毕，单位为second
	//keepTime time.Duration
}

var logger = mylog.GetLogger()

func NewLeaseInfo(aliveTime int64) *LeaseInfo {
	//return &LeaseInfo{aliveTime: aliveTime, keepTime: time.Duration(keepTime) * time.Second}
	return &LeaseInfo{aliveTime: aliveTime}
}

func NewCltConn(lease *LeaseInfo) (*CltConn, error) {
	cfg := clientv3.Config{
		Endpoints:   []string{"101.89.68.10:2379", "101.89.68.11:2379", "101.89.68.12:2379"},
		DialTimeout: 5 * time.Second,
		Context:     context.Background(),
	}
	if client, err := clientv3.New(cfg); err == nil {
		return &CltConn{
			Client: client,
			lease:  lease,
		}, nil
	} else {
		logger.Error("ConnectInit Failed", zap.Error(err))
		return nil, err
	}
}

func (cltConn *CltConn) PutService(key, value string) (*clientv3.PutResponse, error) {
	kv := clientv3.NewKV(cltConn.Client)
	putResp, err := kv.Put(context.Background(), key, value)
	if err != nil {
		logger.Error("Put Operation Failed", zap.Error(err))
		return nil, err
	} else {
		logger.Info("Put Operation Successful")
	}
	return putResp, nil
}

func (cltConn *CltConn) GetService(prefix string) (map[string][]byte, error) {
	kv := clientv3.NewKV(cltConn.Client)
	getResp, err := kv.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		logger.Error("Get Operation Failed", zap.Error(err))
		return nil, err
	} else {
		logger.Info("Get Operation Successful")
	}
	result := make(map[string][]byte)
	for _, value := range getResp.Kvs {
		result[string(value.Key)] = value.Value
	}
	switch prefix {
	case "/test":
		var measureAent conf.MeasureAgent
		measureAent.WriteFile(prefix, result)
	case "/address":
		var address conf.AddressConf
		address.WriteFile(prefix, result)
	case "/agent":
		var agent conf.AgentConf
		agent.WriteFile(prefix, result)
	case "/identity":
		var identity conf.IdentityConf
		identity.WriteFile(prefix, result)
	default:
		logger.Info("it is not to write file")
	}
	return result, nil
}

func (cltConn *CltConn) DelService(prefix string) (*clientv3.DeleteResponse, error) {
	kv := clientv3.NewKV(cltConn.Client)
	delResp, err := kv.Delete(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		logger.Error("Delete Operation Failed", zap.Error(err))
		return nil, err
	} else {
		logger.Info("Get Operation Successful")
	}
	return delResp, nil
}

func (cltConn *CltConn) LeaseService(key, value string) (*clientv3.PutResponse, <-chan *clientv3.LeaseKeepAliveResponse, error) {
	leaseResp, err := cltConn.Client.Grant(cltConn.Client.Ctx(), cltConn.lease.aliveTime)
	kv := clientv3.NewKV(cltConn.Client)
	if err != nil {
		logger.Error("Connect Failed", zap.Error(err))
		return nil, nil, err
	}
	var ctx context.Context
	//if cltConn.lease.keepTime != -1 {
	//	ctx, _ = context.WithTimeout(context.Background(), cltConn.lease.keepTime * time.Second)
	//} else {
	//ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	ctx, _ = context.WithCancel(context.Background())
	//}
	leaseID := leaseResp.ID
	leaseKeepResp, err := cltConn.Client.KeepAlive(ctx, leaseID)
	if err != nil {
		logger.Error("Lease Application Failed", zap.Error(err))
		return nil, nil, err
	} else {
		logger.Info("Lease Operation Successful")
	}
	putResp, err := kv.Put(context.TODO(), key, value, clientv3.WithLease(leaseID))
	if err != nil {
		logger.Error("Put Operation Failed", zap.Error(err))
		return nil, nil, err
	} else {
		logger.Info("Put Operation Successful")
	}
	return putResp, leaseKeepResp, nil
}

//watchKey := cltConn.Client.Watch(context.Background(), prefix, clientv3.WithPrefix())
func (cltConn *CltConn) WatchService(prefix string) {
	watchKey := cltConn.Client.Watch(cltConn.Client.Ctx(), prefix, clientv3.WithPrefix())
	startTime := int64(0)
	for recv := range watchKey {
		logger.Info(string(recv.Events[0].Kv.Key) + " Changed")
		endTime := time.Now().Unix()
		duration := endTime - startTime
		if duration >= 60 {
			_, err := cltConn.GetService(prefix)
			if err != nil {
				return
			}
			startTime = time.Now().Unix()
		}
	}
}

func (cltConn *CltConn) CloseService() {
	err := cltConn.Client.Close()
	if err != nil {
		logger.Error("Connect Close Failed", zap.Error(err))
	} else {
		logger.Info("Connect Close Successful")
	}
}

//测试用的专门用来put数据的一个方法
//func PutTo() {
//	cli, err := clientv3.New(clientv3.Config{
//		Endpoints:   []string{"101.89.68.11:2379"},
//		DialTimeout: 5 * time.Second,
//	})
//	kv := clientv3.NewKV(cli)
//	if err != nil {
//		fmt.Printf("connect is refused, The error is %s", err)
//		return
//	}
//	file, err := os.Open("settings.yml")
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	data, err := ioutil.ReadAll(file)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	var settings conf.Host
//	err = yaml.Unmarshal(data, &settings)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	bytes, err := yaml.Marshal(settings)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	_, err = kv.Put(context.Background(), "/ZhangMing", string(bytes))
//	if err != nil {
//		return
//	}
//	return
//}

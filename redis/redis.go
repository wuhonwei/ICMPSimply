package redis

import (
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"ICMPSimply_bingfa/ICMPSimply/state"
	"fmt"
	"github.com/garyburd/redigo/redis"
)

var logger = mylog.GetLogger()

func Value(address string, key interface{}) (bytes []byte, err error) {
	//conn,err := redis.Dial("tcp","106.3.133.6:6790")
	conn, err := redis.Dial("tcp", address)
	if err != nil {
		//fmt.Println("connect redis error :",err)
		return
	}
	defer func(conn redis.Conn) {
		err = conn.Close()
		if err != nil {
			logger.Error(fmt.Sprintf("close redis error%v\tcpu:%v,mem:%v", err, state.LogCPU, state.LogMEM))
		}
	}(conn)
	bytes, err = redis.Bytes(conn.Do("GET", key))
	if err != nil {
		logger.Error(fmt.Sprintf("get redis bytes error%v\tcpu:%v,mem:%v", err, state.LogCPU, state.LogMEM))
	}
	return
}

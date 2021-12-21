package dao

import (
	"ICMPSimply_bingfa/ICMPSimply/config"
	"ICMPSimply_bingfa/ICMPSimply/dataStore/measureChange"
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"ICMPSimply_bingfa/ICMPSimply/state"
	"ICMPSimply_bingfa/ICMPSimply/statisticsAnalyse"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"strconv"
)

var logger = mylog.GetLogger()

var Db *gorm.DB

func BirthDB(conf *config.Config) (err error) {
	args := fmt.Sprintf("%s:%s@%s(%s)/%s?charset=utf8&parseTime=true&loc=Local", conf.Data.MysqlUser, conf.Data.MysqlPassWord, "tcp", conf.Data.MysqlAddress, "measure-test")
	Db, err = gorm.Open("mysql", args)
	if err != nil {
		//logger.Error("Open "+conf.Data.MysqlAddress+" mysql fail ", zap.Error(err))
		logger.Error(fmt.Sprintf("store\topen %v mysql fail\tcpu:%v,mem:%v", conf.Data.MysqlAddress, state.LogCPU, state.LogMEM))
		return
	}
	return
}
func DeadDB() (err error) {
	if Db != nil {
		err = Db.Close()
		if err != nil {
			//logger.Error("Close mysql fail ", zap.Error(err))
			logger.Error(fmt.Sprintf("store\tClose mysql fail\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		}
	}
	return
}

// TODO 封装各个插件的数据，然后利用GORM上传
func Store(cells []statisticsAnalyse.Cell, conf *config.Config) int {
	if Db == nil {
		err := BirthDB(conf)
		if err != nil {
			return 0
		}
	}
	tx := Db.Begin()
	var affectRows = 0
	for _, item := range cells {
		var err error
		//model
		metric := &measureChange.MetaValue{
			Endpoint: item.Endpoint,
			Metric:   item.Metric,
			Value:    item.Value,
			Step:     strconv.FormatInt(item.Step, 10),
			Type:     item.CounterType,
			//
			Tags:       dictedTagstring(item.Tags),
			Timestamp:  strconv.FormatInt(item.Timestamp, 10),
			SourceIp:   item.SourceIp,
			SourceName: item.SourceName,
			DstIp:      item.DestIp,
		}
		data := measureChange.Convert(metric)

		if !Db.HasTable(data) {
			Db.CreateTable(data)
		}
		if tx.NewRecord(data) {
			err = tx.Create(data).Error
		} else {
			//logger.Error("data exist", zap.Error(err))
			logger.Error(fmt.Sprintf("store\tmeasure data exist\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
		}

		if err == nil {
			affectRows++
		}
	}
	//logger.Info("finish record", zap.Int("nums", affectRows))
	logger.Info(fmt.Sprintf("store\tfinish record measure data\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	tx.Commit()
	//err := tx.Close()
	//if err != nil {
	//	//logger.Error("Close mysql fail ", zap.Error(err))
	//	logger.Error(fmt.Sprintf("store\tClose mysql fail\tcpu:%v,mem:%v", state.LogCPU, state.LogMEM))
	//	return 0
	//}
	return affectRows
}

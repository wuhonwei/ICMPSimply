package conf

import (
	"ICMPSimply_bingfa/ICMPSimply/mylog"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type EtcdWriter interface {
	WriteFile(prefix string, result map[string][]byte)
}

var logger = mylog.GetLogger()

func (m *MeasureAgent) WriteFile(prefix string, result map[string][]byte) {
	m.Points = make([]*Machine, 0, 1)
	for key, value := range result {
		if value == nil {
			continue
		}
		if key == prefix {
			err := yaml.Unmarshal(value, &m.Settings)
			if err != nil {
				logger.Error("Yml Unmarshal Fail", zap.Error(err))
				return
			}
		} else {
			var machine Machine
			err := yaml.Unmarshal(value, &machine)
			if err != nil {
				logger.Error("Yml Unmarshal Fail", zap.Error(err))
				return
			}
			m.Points = append(m.Points, &machine)
		}
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		logger.Error("Yml Marshal Fail", zap.Error(err))
		return
	}
	err = ioutil.WriteFile("/root/sxs_test/MeasureAgent_main.yml", data, os.ModePerm)
	//err = ioutil.WriteFile("settings.yml", data, os.ModePerm)
	if err != nil {
		logger.Error("Write File Failed", zap.Error(err))
		return
	} else {
		logger.Info("Write File Successful")
	}
	return
}

func (a *AddressConf) WriteFile(prefix string, result map[string][]byte) {
	err := ioutil.WriteFile("/root/sxs_test/address.yml", result[prefix], os.ModePerm)
	//err := ioutil.WriteFile("address_test.yml", result[prefix], os.ModePerm)
	if err != nil {
		logger.Error("Write File Failed", zap.Error(err))
		return
	} else {
		logger.Info("Write File Successful")
	}
	return
}

func (i *IdentityConf) WriteFile(prefix string, result map[string][]byte) {
	err := ioutil.WriteFile("/root/sxs_test/identity.yml", result[prefix], os.ModePerm)
	if err != nil {
		logger.Error("Write File Failed", zap.Error(err))
		return
	} else {
		logger.Info("Write File Successful")
	}
	return
}

func (a *AgentConf) WriteFile(prefix string, result map[string][]byte) {
	err := ioutil.WriteFile("/root/sxs_test/agent.yml", result[prefix], os.ModePerm)
	if err != nil {
		logger.Error("Write File Failed", zap.Error(err))
		return
	} else {
		logger.Info("Write File Successful")
	}
	return
}

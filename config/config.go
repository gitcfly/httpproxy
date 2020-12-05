package config

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

var (
	Config *Conf
)

type Conf struct {
	ServerHost    string         `json:"server_host,omitempty"`
	ServerTcpPort int            `json:"server_tcp_port,omitempty"`
	ClientMapping map[string]int `json:"port_mapping,omitempty"`
	LocHttpPort   int            `json:"loc_http_port,omitempty"`
}

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		TimestampFormat:        "2006-01-02 15:04:05",
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	logrus.AddHook(NewContextHook())
	logrus.SetOutput(os.Stdout)
	data, _ := ioutil.ReadFile("./config.json")
	json.Unmarshal(data, &Config)
}

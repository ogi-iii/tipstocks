package utils

import (
	"log"

	"gopkg.in/ini.v1"
)

// Configs : settings from app/config.ini
type Configs struct {
	ServerPort   int
	ServerDebug  bool
	DBPort       int
	DBName       string
	DBCollection string
}

// Conf : contains Configs
var Conf Configs

func init() {
	cfg, cfgErr := ini.Load("app/utils/config.ini")
	if cfgErr != nil {
		log.Fatalln("Cannot load config.ini: ", cfgErr)
		return
	}
	Conf = Configs{
		ServerPort:   cfg.Section("server").Key("port").MustInt(50051),
		ServerDebug:  cfg.Section("server").Key("port").MustBool(true),
		DBPort:       cfg.Section("db").Key("port").MustInt(27017),
		DBName:       cfg.Section("db").Key("name").String(),
		DBCollection: cfg.Section("db").Key("collection").String(),
	}
}

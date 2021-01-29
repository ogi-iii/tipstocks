package utils

import (
	"log"

	"gopkg.in/ini.v1"
)

// Configs : settings from app/config.ini
type Configs struct {
	ServerPort   int
	ServerDebug  bool
	ClientPort   int
	ClientDebug  bool
	DBPort       int
	DBName       string
	DBCollection string
}

// Conf : contains Configs
var Conf Configs

// LoadConf : load settings from config.ini
func LoadConf(path string) Configs {
	cfg, cfgErr := ini.Load(path)
	if cfgErr != nil {
		log.Fatalln("Cannot load config.ini: ", cfgErr)
	}
	return Configs{
		ServerPort:   cfg.Section("server").Key("port").MustInt(50051),
		ServerDebug:  cfg.Section("server").Key("debug").MustBool(true),
		ClientPort:   cfg.Section("client").Key("port").MustInt(8000),
		ClientDebug:  cfg.Section("client").Key("debug").MustBool(true),
		DBPort:       cfg.Section("db").Key("port").MustInt(27017),
		DBName:       cfg.Section("db").Key("name").String(),
		DBCollection: cfg.Section("db").Key("collection").String(),
	}
}

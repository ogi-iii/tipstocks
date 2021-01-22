package setting

import (
	"log"

	"gopkg.in/ini.v1"
)

// Settings : settings from app/config.ini
type Settings struct {
	ServerPort   int
	ServerDebug  bool
	DBPort       int
	DBName       string
	DBCollection string
}

// Conf : contains Settings
var Conf Settings

func init() {
	cfg, cfgErr := ini.Load("app/setting/config.ini")
	if cfgErr != nil {
		log.Fatalln("Cannot load config.ini: ", cfgErr)
		return
	}
	Conf = Settings{
		ServerPort:   cfg.Section("server").Key("port").MustInt(50051),
		ServerDebug:  cfg.Section("server").Key("port").MustBool(true),
		DBPort:       cfg.Section("db").Key("port").MustInt(27017),
		DBName:       cfg.Section("db").Key("name").String(),
		DBCollection: cfg.Section("db").Key("collection").String(),
	}
}

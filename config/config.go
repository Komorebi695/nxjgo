package config

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	nxjLog "github.com/Komorebi695/nxjgo/log"
	"os"
)

var Conf = &NXJConfig{
	logger:   nxjLog.Default(),
	Log:      make(map[string]any),
	Template: make(map[string]any),
	Pool:     make(map[string]any),
}

type NXJConfig struct {
	logger   *nxjLog.Logger
	Log      map[string]any
	Template map[string]any
	Pool     map[string]any
}

//func init() {
//	localToml()
//}

func localToml() {
	configFile := flag.String("conf", "conf/app.toml", "app config file")
	flag.Parse()
	if _, err := os.Stat(*configFile); err != nil {
		Conf.logger.Error(fmt.Sprintf("%v file load fail,because not exist", *configFile))
		return
	}
	_, err := toml.DecodeFile(*configFile, Conf)
	if err != nil {
		Conf.logger.Error(fmt.Sprintf("%v decode fail check format", *configFile))
		return
	}
	return
}

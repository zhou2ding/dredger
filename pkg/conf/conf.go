package conf

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var Conf *viper.Viper

func InitConf(configPath string) {
	Conf = viper.New()
	if len(configPath) == 0 {
		Conf.SetConfigName("road")
		Conf.SetConfigType("yaml")
		Conf.AddConfigPath("./") //常规部署
	} else {
		Conf.SetConfigFile(configPath)
	}
	//设置配置默认值
	setDefaultConfig(Conf)
	if err := Conf.ReadInConfig(); err != nil {
		fmt.Printf("read config error，file path %s\n", Conf.ConfigFileUsed())
		panic(err)
	}
	Conf.WatchConfig()
	Conf.OnConfigChange(func(in fsnotify.Event) {})
}

func setDefaultConfig(v *viper.Viper) {
	v.SetDefault("log.path", "../../build/test")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.size", 10)
	v.SetDefault("log.expire", 3)
	v.SetDefault("log.limit", 15)
	v.SetDefault("log.stdout", true)
}

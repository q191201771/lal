package main

import (
	"flag"
	"fmt"
	"github.com/q191201771/nezha/pkg/bininfo"
	"github.com/q191201771/nezha/pkg/log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

var sm *ServerManager

func main() {
	confFile := parseFlag()

	initLog()

	log.Infof("bininfo: %s", bininfo.StringifySingleLine())

	config := loadConf(confFile)

	sm = NewServerManager(config)
	go sm.RunLoop()

	//shutdownAfter(60 * time.Second)

	// TODO chef: 添加优雅退出信号处理

	startWebPProf()
}

func parseFlag() string {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	flag.Parse()
	if *binInfoFlag {
		fmt.Fprintln(os.Stderr, bininfo.StringifyMultiLine())
		os.Exit(1)
	}
	if *cf == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *cf
}

func initLog() {
	// TODO chef: 在配置文件中配置这些
	c := log.Config{
		Level:       log.LevelDebug,
		Filename:    "./logs/lal.log",
		IsToStdout:  true,
		RotateMByte: 1024,
	}
	if err := log.Init(c); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initial log failed. err=%v", err)
		os.Exit(1)
	}
	log.Info("initial log succ.")
}

func loadConf(confFile string) *Config {
	config, err := LoadConf(confFile)
	if err != nil {
		log.Errorf("load Conf failed. file=%s err=%v", confFile, err)
		os.Exit(1)
	}
	log.Infof("load conf file succ. file=%s content=%v", confFile, config)
	return config
}

func startWebPProf() {
	if err := http.ListenAndServe("0.0.0.0:10001", nil); err != nil {
		log.Error(err)
		return
	}
	log.Info("start pprof listen. addr=:10001")
}

func shutdownAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		sm.Dispose()
	}()
}

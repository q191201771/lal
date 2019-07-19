package main

import (
	"flag"
	"fmt"
	"github.com/q191201771/lal/pkg/util/bininfo"
	"github.com/q191201771/lal/pkg/util/log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

var sm *ServerManager

func main() {
	confFile, logConfFile := parseFlag()

	initLog(logConfFile)

	log.Infof("bininfo: %s", bininfo.StringifySingleLine())

	config := loadConf(confFile)

	sm = NewServerManager(config)
	go sm.RunLoop()

	//shutdownAfter(60 * time.Second)

	// TODO chef: 添加优雅退出信号处理

	startWebPProf()
}

func parseFlag() (string, string) {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	lcf := flag.String("l", "", "specify log conf file")
	flag.Parse()
	if *binInfoFlag {
		fmt.Fprintln(os.Stderr, bininfo.StringifyMultiLine())
		os.Exit(1)
	}
	if *cf == "" || *lcf == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *cf, *lcf
}

func initLog(logConfFile string) {
	if err := log.Initial(logConfFile); err != nil {
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

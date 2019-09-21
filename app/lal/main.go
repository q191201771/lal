package main

import (
	"flag"
	"fmt"
	"github.com/q191201771/nezha/pkg/bininfo"
	"github.com/q191201771/nezha/pkg/log"
	"github.com/q191201771/lal/pkg/logic"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

var sm *logic.ServerManager

func main() {
	confFile := parseFlag()
	config := loadConf(confFile)
	initLog(config.Log)
	log.Infof("bininfo: %s", bininfo.StringifySingleLine())

	sm = logic.NewServerManager(config)

	if config.PProf.Addr != "" {
		go runWebPProf(config.PProf.Addr)
	}
	go runSignalHandler()

	sm.RunLoop()
}

func parseFlag() string {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	flag.Parse()
	if *binInfoFlag {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		os.Exit(0)
	}
	if *cf == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *cf
}

func loadConf(confFile string) *logic.Config {
	config, err := logic.LoadConf(confFile)
	if err != nil {
		log.Errorf("load conf failed. file=%s err=%+v", confFile, err)
		os.Exit(1)
	}
	log.Infof("load conf file succ. file=%s content=%+v", confFile, config)
	return config
}

func initLog(config log.Config) {
	if err := log.Init(config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initial log failed. err=%+v", err)
		os.Exit(1)
	}
	log.Info("initial log succ.")
}

func runWebPProf(addr string) {
	log.Infof("start web pprof listen. addr=%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Error(err)
		return
	}
}

func runSignalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)
	s := <-c
	log.Infof("recv signal. s=%+v", s)
	sm.Dispose()
}

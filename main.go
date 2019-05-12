package main

import (
	"flag"
	"fmt"
	"github.com/q191201771/lal/log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

var config *Config
var manager *Manager

func main() {
	confFile := flag.String("c", "", "specify conf file")
	logConfFile := flag.String("l", "", "specify log conf file")
	flag.Parse()
	if *confFile == "" || *logConfFile == "" {
		flag.Usage()
		return
	}

	if err := log.Initial(*logConfFile); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initial log failed. err=%v", err)
		return
	}
	log.Info("initial log succ.")

	config, err := LoadConf(*confFile)
	if err != nil {
		log.Errorf("load Conf failed. file=%s err=%v", *confFile, err)
		return
	}
	log.Infof("load conf file succ. file=%s content=%v", *confFile, config)

	manager = NewManager(config)

	go manager.RunLoop()

	//shutdownAfter(60 * time.Second)

	if err := http.ListenAndServe("0.0.0.0:10001", nil); err != nil {
	}
}

func shutdownAfter(d time.Duration) {
	go func() {
		time.Sleep(d)
		manager.Dispose()
	}()
}

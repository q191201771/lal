// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
	log "github.com/q191201771/naza/pkg/nazalog"
)

var sm *logic.ServerManager

func main() {
	confFile := parseFlag()
	config := loadConf(confFile)
	initLog(config.Log)
	log.Infof("bininfo: %s", bininfo.StringifySingleLine())

	sm = logic.NewServerManager(&config.Config)

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
		_, _ = fmt.Fprintf(os.Stderr, `
Example:
  ./bin/lals -c ./conf/lals.conf.json
`)
		os.Exit(1)
	}
	return *cf
}

func loadConf(confFile string) *Config {
	config, err := LoadConf(confFile)
	if err != nil {
		log.Errorf("load conf failed. file=%s err=%+v", confFile, err)
		os.Exit(1)
	}
	log.Infof("load conf file succ. file=%s content=%+v", confFile, config)
	return config
}

func initLog(opt log.Option) {
	if err := log.Init(func(option *log.Option) {
		*option = opt
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initial log failed. err=%+v\n", err)
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

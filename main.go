package main

import (
	"flag"
	"github.com/q191201771/lal/httpflv"
	"log"
	"net/http"
	_ "net/http/pprof"
)

var config *Config

func main() {
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:10000", nil))
	}()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	confFile := flag.String("c", "", "specify conf file")
	flag.Parse()
	if *confFile == "" {
		flag.Usage()
		return
	}

	config, err := LoadConf(*confFile)
	if err != nil {
		log.Println("Load Conf failed.", confFile, err)
	}
	log.Println("load conf file.", *confFile, *config)

	manager := httpflv.NewManager(config.HttpFlv)

	//go func() {
	//	time.Sleep(60 * time.Second)
	//	manager.Dispose()
	//}()

	manager.RunLoop()
}

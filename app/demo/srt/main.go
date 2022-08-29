package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
	"log"
	"os"
)

func main() {
	confFilename := parseFlag()
	srv := logic.NewLalServer(func(option *logic.Option) {
		option.ConfFilename = confFilename
	})
	
	server := NewServer("0.0.0.0", 6001, srv)
	go server.Run(context.Background())

	if err := srv.RunLoop(); err != nil {
		log.Panic(err)
	}
}

func parseFlag() string {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	flag.Parse()

	if *binInfoFlag {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		_, _ = fmt.Fprintln(os.Stderr, base.LalFullInfo)
		os.Exit(0)
	}
	return *cf
}

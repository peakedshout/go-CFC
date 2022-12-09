package main

import (
	"flag"
	"github.com/peakedshout/go-CFC/server"
	"github.com/peakedshout/go-CFC/tool"
)

func main() {
	path := flag.String("c", "./config.json", `Default configuration file location. If not specified, the default is "./config.json".`)
	flag.Parse()
	config := tool.GetCFCHookConfig(*path)
	server.NewServer(config.Ct.IP, config.Ct.Port, config.Ct.Key)
}

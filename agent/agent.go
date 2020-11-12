package main

import (
	"bgp_agent/agent/app"
	_ "bgp_agent/utils/log"
	"bgp_agent/utils/procinfo"
	"bgp_agent/utils/rest"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

var (
	hostname string
	configfile string
	bgpconfigfile string
)
func init()  {
	flag.StringVar(&hostname, "hostname", "localhost", "bgp peer router-id")
	flag.StringVar(&bgpconfigfile, "bgpconfigfile", "/export/log/bgp_agent/server.toml", "show bgp config")
	flag.StringVar(&configfile, "configfile", "/export/log/bgp_agent/server.toml", "show proc config")
}

func main() {

	flag.Parse()

	app, err := app.New(hostname, bgpconfigfile, configfile)
	if err != nil{
		logrus.Errorf("startup bgp agent failed: %v", err)
		return
	}
	app.Runs()

	gin.SetMode(gin.DebugMode)
	engine := gin.New()

	proc := engine.Group("/v1").Use(rest.RequestPrint)
	{
		proc.GET("/gitinfo", procinfo.GitInfo)
	}
	engine.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))

	port := os.Getenv("agent_port")
	if len(port) == 0{
		port = "8889"
	}
	url := fmt.Sprintf("0.0.0.0:%s", port)
	err = engine.Run(url)
	logrus.Errorf("restAPI server run failed: %v. \n", err)
}


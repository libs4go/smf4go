package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/libs4go/scf4go"
	"github.com/libs4go/scf4go/reader/file"
	"github.com/libs4go/slf4go"
	"github.com/libs4go/smf4go"
)

func isDir(path string) bool {
	fi, err := os.Stat(path)

	if err != nil {
		return false
	}

	return fi.IsDir()
}

// Run start a smf4go app
func Run(appname string) {
	configpath := flag.String("config", fmt.Sprintf("./%s.json", appname), "special the mesh app config file")

	flag.Parse()

	config := scf4go.New()

	if isDir(*configpath) {
		if err := config.Load(file.New(file.Dir(*configpath))); err != nil {
			println(fmt.Sprintf("load config from directory %s %s", *configpath, err))
			return
		}
	} else {
		if err := config.Load(file.New(file.File(*configpath))); err != nil {
			println(fmt.Sprintf("load config file %s %s", *configpath, err))
			return
		}
	}

	if err := slf4go.Config(config.SubConfig("slf4go")); err != nil {
		println(fmt.Sprintf("set slf4go config error: %s", err))
		return
	}

	logger := slf4go.Get(appname)
	defer slf4go.Sync()

	logger.I("start app {@app}", appname)

	if err := smf4go.Builder().Start(config); err != nil {
		logger.I("start gomesh error: \n{@err}", err)
		return
	}

	logger.I("start app {@app} -- success", appname)

	<-make(chan int)
}

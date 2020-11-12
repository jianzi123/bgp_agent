package log

import (
	"fmt"
	"github.com/orandin/lumberjackrus"
	"github.com/sirupsen/logrus"
	"path"
	"runtime"
)

const (
	LOGFILE      = "/export/log/bgp_agent/bgp_agent.log"
	LOGINFOFILE  = "/export/log/bgp_agent/bgp_agent_info.log"
	LOGERRORFILE = "/export/log/bgp_agent/bgp_agent_error.log"
	LOGPANICFILE = "/export/log/bgp_agent/bgp_agent_panic.log"
	LOGDEBUGFILE = "/export/log/bgp_agent/bgp_agent_debug.log"
)

func init() {
	// 设置日志输出格式
	// https://github.com/sirupsen/logrus/issues/63
	logrus.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	})
	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.DebugLevel)
	// set log file
	// https://github.com/orandin/lumberjackrus
	// https://github.com/sirupsen/logrus/issues/678
	// https://github.com/sirupsen/logrus/issues/877
	hook, err := lumberjackrus.NewHook(
		&lumberjackrus.LogFile{
			Filename:   LOGFILE,
			MaxSize:    100,
			MaxBackups: 1,
			MaxAge:     1,
			//Compress:   false,
			//LocalTime:  false,
		},
		logrus.DebugLevel,
		&logrus.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				filename := path.Base(f.File)
				return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
			},
		},
		&lumberjackrus.LogFileOpts{
			logrus.DebugLevel: &lumberjackrus.LogFile{
				Filename:   LOGDEBUGFILE,
				MaxSize:    100,   // optional
				MaxBackups: 1,     // optional
				MaxAge:     1,     // optional
				Compress:   false, // optional
				LocalTime:  false, // optional
			},
			logrus.InfoLevel: &lumberjackrus.LogFile{
				Filename:   LOGINFOFILE,
				MaxSize:    100,   // optional
				MaxBackups: 1,     // optional
				MaxAge:     1,     // optional
				Compress:   false, // optional
				LocalTime:  false, // optional
			},
			logrus.ErrorLevel: &lumberjackrus.LogFile{
				Filename:   LOGERRORFILE,
				MaxSize:    100,   // optional
				MaxBackups: 1,     // optional
				MaxAge:     1,     // optional
				Compress:   false, // optional
				LocalTime:  false, // optional
			},
			logrus.PanicLevel: &lumberjackrus.LogFile{
				Filename: LOGPANICFILE,
			},
		},
	)

	if err != nil {
		logrus.Errorf("set loghook failed: %v. \n", err)
	}

	logrus.AddHook(hook)
}


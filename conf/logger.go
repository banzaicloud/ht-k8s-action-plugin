package conf

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *logrus.Logger

func Logger() *logrus.Logger {
	if logger == nil {
		logger = logrus.New()
		switch viper.GetString("log.level") {
		case "debug":
			logrus.SetLevel(logrus.DebugLevel)
		case "info":
			logrus.SetLevel(logrus.InfoLevel)
		case "warn":
			logrus.SetLevel(logrus.WarnLevel)
		case "error":
			logrus.SetLevel(logrus.ErrorLevel)
		case "fatal":
			logrus.SetLevel(logrus.FatalLevel)
		default:
			logrus.SetLevel(logrus.InfoLevel)
			logrus.Warnf("Invalid log level: %s. Defaulting to info.\n", viper.GetString("log.level"))
		}

		switch viper.GetString("log.format") {
		case "text":
			logrus.SetFormatter(new(logrus.TextFormatter))
		case "json":
			logrus.SetFormatter(new(logrus.JSONFormatter))
		default:
			logrus.SetFormatter(new(logrus.TextFormatter))
			logrus.Warnf("Invalid log format: %s. Defaulting to text.\n", viper.GetString("log.format"))
		}

		logger.Infof("Configured logger: log.level=%s, log.format=%s", logrus.GetLevel(), viper.GetString("log.format"))
	}
	return logger
}

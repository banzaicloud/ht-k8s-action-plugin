package plugin

import (
	as "github.com/banzaicloud/hollowtrees/actionserver"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/ht-k8s-action-plugin/conf"
	"fmt"
)

var log *logrus.Entry

func init() {
	log = conf.Logger().WithField("package", "plugin")
}

func RouteEvent(event *as.AlertEvent) error {
	switch event.EventType {
	case "prometheus.server.alert.SpotTerminationNotice":
		fmt.Println("I got a spot termination notice")
	}
	return nil
}

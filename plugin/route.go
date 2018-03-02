package plugin

import (
	as "github.com/banzaicloud/hollowtrees/actionserver"
	log "github.com/sirupsen/logrus"
)

type EventRouter struct {
	configRoot string
}

func NewEventRouter(cr string) *EventRouter {
	return &EventRouter{
		configRoot: cr,
	}
}

func (r *EventRouter) RouteEvent(event *as.AlertEvent) error {
	log.Infof("Received %s", event.EventType)
	switch event.EventType {
	case "prometheus.server.alert.SpotTerminationNotice":
		d, err := NewDrainer(r.configRoot, event.Data["cluster_name"])
		if err != nil {
			log.Errorf("Couldn't create drainer: %s", err.Error())
			return err
		}
		err = d.DrainNode(event.Data["instance"])
		if err != nil {
			log.Errorf("Failed to drain node: %s", err.Error())
			return err
		}
	}
	return nil
}

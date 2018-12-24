package main

import (
	"encoding/json"
	ev "github.com/asaskevich/EventBus"
	"log"
)

type Logger func(interface{})

func subscribeAndLog(bus ev.Bus, topic string) {
	if err := bus.Subscribe(topic, func(msg interface{}) {
		if toLog, e := json.Marshal(struct {
			topic string
			msg   interface{}
		}{
			topic: topic,
			msg:   msg,
		}); e != nil {
			println(e)
		} else {
			println(string(toLog))
		}
	}); err != nil {
		log.Fatal(err)
	}
}

func StartEventLogger(bus ev.Bus) {
	subscribeAndLog(bus, MailSubmitted)
}

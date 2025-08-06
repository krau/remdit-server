package service

import (
	"github.com/duke-git/lancet/v2/eventbus"
)

type Event struct {
	Type   string
	FileID string
	Data   any
	Err    error
}

var eventBus = eventbus.NewEventBus[Event]()

func PublishEvent(eventType string, fileID string, data any, err error) {
	event := Event{
		Type:   eventType,
		FileID: fileID,
		Data:   data,
		Err:    err,
	}
	eventBus.Publish(eventbus.Event[Event]{Topic: fileID, Payload: event})
}

func SubscribeEvent(fileID string, handler func(eventData Event)) {
	eventBus.Subscribe(fileID, handler, true, 0, nil)
}

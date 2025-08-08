package service

import (
	"github.com/duke-git/lancet/v2/eventbus"
)

type Event struct {
	Type   string
	FileID string
	Data   any
}

type EventFileSaveStatusData struct {
	Ok  bool
	Err string
}

var eventBus = eventbus.NewEventBus[Event]()

func PublishEvent(eventType string, fileID string, data any) {
	event := Event{
		Type:   eventType,
		FileID: fileID,
		Data:   data,
	}
	eventBus.Publish(eventbus.Event[Event]{Topic: fileID, Payload: event})
}

func SubscribeEvent(fileID string, handler func(eventData Event), filter func(eventData Event) bool) {
	eventBus.Subscribe(fileID, handler, false, 0, filter)
}

func UnsubscribeEvent(fileID string, handler func(eventData Event)) {
	eventBus.Unsubscribe(fileID, handler)
}

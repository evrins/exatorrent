package core

import (
	"encoding/json"
	"github.com/nsqio/go-nsq"
)

var eventProducer *nsq.Producer

func InitEventProducer(address string) {
	var err error
	eventProducer, err = nsq.NewProducer(address, nsq.NewConfig())
	if err != nil {
		Err.Printf("fail to init producer. err: %v", err)
		return
	}
	Info.Printf("nsq %s connected", address)
}

type EventType string

var (
	Empty     EventType = "empty"
	Added     EventType = "added"
	Loaded    EventType = "loaded"
	Started   EventType = "started"
	Stopped   EventType = "stopped"
	Removed   EventType = "removed"
	Deleted   EventType = "deleted"
	Completed EventType = "completed"
)

type Event struct {
	Hash      string    `json:"hash"`
	EventType EventType `json:"eventType"`
}

var topic = "torrent"

func PublishEvent(hash string, eventType EventType) {
	if eventProducer == nil {
		Warn.Println("nsq not connected won't publish event")
		return
	}

	var event = Event{
		Hash:      hash,
		EventType: eventType,
	}
	buf, err := json.Marshal(event)
	if err != nil {
		Warn.Printf("fail to marshal event. err: %v", err)
		return
	}

	err = eventProducer.Publish(topic, buf)
	if err != nil {
		Warn.Println("fail to publish event. err: %v", err)
		return
	}
}

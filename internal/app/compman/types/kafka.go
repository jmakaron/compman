package types

import (
	"encoding/json"
	"errors"
)

const (
	opInsert = "insert"
	opUpdate = "update"
	opDelete = "delete"
)

var (
	ErrUnsupportedOperation = errors.New("unsupported operation")
	cmdTopic                = "commandTopic"
	eventTopic              = map[string]string{opInsert: cmdTopic, opUpdate: cmdTopic, opDelete: cmdTopic}
)

type KafkaCompanyEvent struct {
	topic *string
	*Company
	Op string `json:"op"`
}

func (e *KafkaCompanyEvent) Topic() *string {
	return e.topic
}

func (e *KafkaCompanyEvent) Key() []byte {
	return []byte(e.ID)
}

func (e *KafkaCompanyEvent) Value() []byte {
	b, _ := json.Marshal(e)
	return b
}

func NewKafkaCompanyEvent(c *Company, op string) (*KafkaCompanyEvent, error) {
	if op != opInsert && op != opUpdate && op != opDelete {
		return nil, ErrUnsupportedOperation
	}
	var topic string
	rv := KafkaCompanyEvent{Company: c, Op: op}
	topic = eventTopic[op]
	rv.topic = &topic
	return &rv, nil
}

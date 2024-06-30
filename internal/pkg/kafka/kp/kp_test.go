package kp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-test/deep"
	"github.com/google/uuid"
	"github.com/jmakaron/compman/internal/app/compman/types"
)

func TestKafkaIntegration(t *testing.T) {
	cfg := ProducerCfg{
		BootstrapServers:    "127.0.0.1:9092",
		Acks:                -1,
		EnableIdempotence:   false,
		MaxInFlight:         5,
		Retries:             5,
		QueueBufferingMaxMs: 100,
		CompressionCodec:    "none",
		Debug:               ",",
	}
	p := New(cfg)
	ctx := context.Background()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("failed to connect producer to kafka, %+v", err)
	}
	descCoop, descNonProfit := "cooperative company description", "non-profit company description"
	data := []*types.Company{
		&types.Company{
			ID:    uuid.NewString(),
			Name:  "sole-proprietorship-name-1",
			CType: types.CompanyTypeSoleProprietorship,
		},
		&types.Company{
			ID:          uuid.NewString(),
			Name:        "cooperative-name-1",
			Desc:        &descCoop,
			EmployeeCnt: 1337,
			Registered:  true,
			CType:       types.CompanyTypeCooperative,
		},
		&types.Company{
			ID:          uuid.NewString(),
			Name:        "non-profit-name-1",
			Desc:        &descNonProfit,
			EmployeeCnt: 10000,
			Registered:  true,
			CType:       types.CompanyTypeNonProfit,
		},
		&types.Company{
			ID:          uuid.NewString(),
			Name:        "corporation-name-1",
			EmployeeCnt: 1000,
			Registered:  true,
			CType:       types.CompanyTypeCorporation,
		},
		&types.Company{
			ID:          uuid.NewString(),
			Name:        "sole-proprietorship-name-2",
			EmployeeCnt: 1,
			Registered:  true,
			CType:       types.CompanyTypeSoleProprietorship,
		},
		&types.Company{
			ID:          uuid.NewString(),
			Name:        "corporation-name-2",
			EmployeeCnt: 100000,
			Registered:  true,
			CType:       types.CompanyTypeCorporation,
		},
	}
	evts := make([]KEvent, len(data))
	for i, d := range data {
		var op string
		if i < 3 {
			op = "insert"
		} else if i < 5 {
			op = "update"
		} else {
			op = "delete"
		}
		var e *types.KafkaCompanyEvent
		e, _ = types.NewKafkaCompanyEvent(d, op)
		evts[i] = e
	}
	if err := p.PublishWithRetry(evts...); err != nil {
		t.Fatalf("failed to publish messages to kafka with retries, %+v", err)
	}
	defer p.Disconnect()
	//	fmt.Fprintf(os.Stderr, "disconnected\n")
	config := kafka.ConfigMap{
		kafkaCFGBootstrapServers: "127.0.0.1:9092",
		"group.id":               "test_group",
		"auto.offset.reset":      "earliest",
	}

	fmt.Fprintf(os.Stderr, "creating consumer\n")
	kc, err := kafka.NewConsumer(&config)
	if err != nil {
		t.Fatalf("failed to create consumer, %+v", err)
	}
	received := map[string]*types.KafkaCompanyEvent{}

	if err = kc.Subscribe("commandTopic", nil); err != nil {
		t.Fatalf("failed to subscribe to 'commandTopic' topic, %+v", err)
	}

	for {
		e, err := kc.ReadMessage(500 * time.Millisecond)
		if err != nil {
			var kErr kafka.Error
			if errors.As(err, &kErr) {
				if kErr.IsFatal() {
					kc.Unsubscribe()
					kc.Close()
					t.Fatalf("received fatal error from kafka, %+v", err)
				}
			}
		}
		if e.TopicPartition.Error != nil {
			t.Fatalf("received topic partition error from kafka, %+v", e.TopicPartition.Error)
		}
		var evt types.KafkaCompanyEvent
		if err = json.Unmarshal(e.Value, &evt); err != nil {
			kc.Unsubscribe()
			kc.Close()
			t.Fatalf("could not unmarshal kafka message, %+v", err)
		}
		fmt.Fprintf(os.Stderr, "received message %s %+v\n", string(e.Key), evt)
		received[string(e.Key)] = &evt
		kc.StoreMessage(e)

		if len(received) == len(evts) {
			break
		}
	}
	kc.Unsubscribe()
	kc.Close()

	for idx, c := range data {
		var op string
		if idx < 3 {
			op = "insert"
		} else if idx < 5 {
			op = "update"
		} else {
			op = "delete"
		}
		if e, ok := received[c.ID]; !ok {
			t.Errorf("expected to find message with id '%s' but didnt find it", c.ID)
		} else {
			if e.Op != op {
				t.Errorf("expected operation '%s', got '%s'", op, e.Op)
			}
			if diff := deep.Equal(c, e.Company); diff != nil {
				t.Errorf("expected company %+v, but got company %+v", c, e.Company)
			}
		}
	}
}

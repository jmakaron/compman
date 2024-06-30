package kp

import (
	"context"
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const (
	maxRetries    = 3
	backoffFactor = 2
	baseDelay     = 100 * time.Millisecond
)

type KEvent interface {
	Topic() *string
	Key() []byte
	Value() []byte
}

const (
	kafkaCFGBootstrapServers    string = "bootstrap.servers"
	kafkaCFGAcks                string = "acks"
	kafkaCFGEnableIdempotence   string = "enable.idempotence"
	kafkaCFGMaxInFlight         string = "max.in.flight"
	kafkaCFGRetries             string = "retries"
	kafkaCFGQueueBufferingMaxMs string = "queue.buffering.max.ms"
	kafkaCFGCompressionCodec    string = "compression.codec"
	kafkaCFGDebug               string = "debug"
)

type ProducerCfg struct {
	BootstrapServers    string `json:"bootstrap_servers"`
	Acks                int    `json:"acks"`
	EnableIdempotence   bool   `json:"enable_idempotence"`
	MaxInFlight         int    `json:"max_in_flight"`
	Retries             int    `json:"retries"`
	QueueBufferingMaxMs int    `json:"queue_buffering_max_ms"`
	CompressionCodec    string `json:"compression_codec"`
	Debug               string `json:"debug"`
}

type KafkaProducer interface {
	Connect(context.Context) error
	Disconnect()
	Publish(...KEvent) error
	PublishWithRetry(...KEvent) error
}

type kafkaProducer struct {
	cfg    ProducerCfg
	p      *kafka.Producer
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *kafkaProducer) Connect(ctx context.Context) error {
	conf := kafka.ConfigMap{
		kafkaCFGBootstrapServers:    p.cfg.BootstrapServers,
		kafkaCFGAcks:                p.cfg.Acks,
		kafkaCFGEnableIdempotence:   p.cfg.EnableIdempotence,
		kafkaCFGMaxInFlight:         p.cfg.MaxInFlight,
		kafkaCFGRetries:             p.cfg.Retries,
		kafkaCFGQueueBufferingMaxMs: p.cfg.QueueBufferingMaxMs,
		kafkaCFGCompressionCodec:    p.cfg.CompressionCodec,
		kafkaCFGDebug:               p.cfg.Debug,
	}
	var err error
	p.p, err = kafka.NewProducer(&conf)
	if err != nil {
		return err
	}
	p.ctx, p.cancel = context.WithCancel(ctx)
	return nil
}

func (p *kafkaProducer) PublishWithRetry(evts ...KEvent) error {
	var err error
	for r := 0; r < maxRetries; r++ {
		if err != nil {
			exp := 1 << uint(r-1)
			currentDelay := time.Duration(float64(baseDelay) * (backoffFactor - 1.0) * float64(exp))
			time.Sleep(currentDelay)
		}
		if err = p.Publish(evts...); err == nil {
			break
		}
	}
	return err
}

func (p *kafkaProducer) Publish(evts ...KEvent) error {
	respCh := make(chan kafka.Event, len(evts))
	errCh := make(chan error, len(evts))
	var err error
	var total int
	for idx, e := range evts {
		select {
		case <-p.ctx.Done():
			err = p.ctx.Err()
		default:
			err = p.p.Produce(
				&kafka.Message{TopicPartition: kafka.TopicPartition{
					Partition: kafka.PartitionAny, Topic: e.Topic()}, Value: e.Value(), Key: e.Key()},
				respCh)
		}
		if err != nil {
			total = idx
			break
		}
	}
	if err == nil {
		total = len(evts)
	}

	go func(total int) {
		defer close(errCh)
		for i := 0; i < total; i++ {
			e := <-respCh
			select {
			case <-p.ctx.Done():
				return
			default:
				var err error
				switch t := e.(type) {
				case kafka.Error:
					err = t
				case *kafka.Message:
					err = t.TopicPartition.Error
				}
				if err != nil {
					errCh <- err
				}
			}
		}
	}(total)

	for e := range errCh {
		if e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}

func (p *kafkaProducer) Disconnect() {
	p.cancel()
	rem := p.p.Flush(5000)
	for i := 0; i < rem; i++ {
		<-p.p.Events()
	}
	p.p.Close()
}

func New(cfg ProducerCfg) KafkaProducer {
	return &kafkaProducer{cfg: cfg}
}

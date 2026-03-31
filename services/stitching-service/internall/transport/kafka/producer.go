package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
	"kseilons/stitching-service/internall/domain"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	return &Producer{
		writer: writer,
	}
}

func (p *Producer) ReportResult(ctx context.Context, result domain.RenderResult) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(result.TaskID),
		Value: payload,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

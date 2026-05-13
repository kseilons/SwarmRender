package kafka

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"kseilons/worker-service/internall/domain"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestResultProducer_ReportResult_Success(t *testing.T) {
	mockWriter := new(MockWriter)
	logger := slog.Default()
	p := &ResultProducer{
		writer: mockWriter,
		logger: logger,
	}

	ctx := context.Background()
	result := domain.RenderResult{
		TaskID: "task-1",
		Status: domain.StatusCompleted,
	}

	mockWriter.On("WriteMessages", ctx, mock.MatchedBy(func(msgs []kafka.Message) bool {
		return len(msgs) == 1 && string(msgs[0].Key) == "task-1"
	})).Return(nil).Once()

	err := p.ReportResult(ctx, result)
	assert.NoError(t, err)
	mockWriter.AssertExpectations(t)
}

func TestResultProducer_ReportResult_WriteError(t *testing.T) {
	mockWriter := new(MockWriter)
	p := &ResultProducer{
		writer: mockWriter,
		logger: slog.Default(),
	}

	ctx := context.Background()
	result := domain.RenderResult{TaskID: "task-1"}

	mockWriter.On("WriteMessages", ctx, mock.Anything).Return(errors.New("kafka error")).Once()

	err := p.ReportResult(ctx, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write result to kafka")
}

func TestResultProducer_Close(t *testing.T) {
	mockWriter := new(MockWriter)
	p := &ResultProducer{writer: mockWriter}

	mockWriter.On("Close").Return(nil).Once()
	err := p.Close()
	assert.NoError(t, err)
}

func TestNewResultProducer(t *testing.T) {
	p := NewResultProducer([]string{"localhost:9092"}, "topic", slog.Default())
	assert.NotNil(t, p)
	assert.NotNil(t, p.writer)
}

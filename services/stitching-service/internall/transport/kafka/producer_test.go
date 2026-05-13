package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"kseilons/stitching-service/internall/domain"
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

func TestProducer_ReportResult_Success(t *testing.T) {
	mockWriter := new(MockWriter)
	p := &Producer{writer: mockWriter}
	ctx := context.Background()
	result := domain.RenderResult{TaskID: "task-1", Status: domain.StatusCompleted}

	mockWriter.On("WriteMessages", ctx, mock.Anything).Return(nil).Once()
	err := p.ReportResult(ctx, result)
	assert.NoError(t, err)
}

func TestProducer_ReportResult_Error(t *testing.T) {
	mockWriter := new(MockWriter)
	p := &Producer{writer: mockWriter}
	ctx := context.Background()
	result := domain.RenderResult{TaskID: "task-1", Status: domain.StatusCompleted}

	mockWriter.On("WriteMessages", ctx, mock.Anything).Return(errors.New("kafka error")).Once()
	err := p.ReportResult(ctx, result)
	assert.Error(t, err)
}

func TestProducer_Close(t *testing.T) {
	mockWriter := new(MockWriter)
	p := &Producer{writer: mockWriter}
	mockWriter.On("Close").Return(nil).Once()
	err := p.Close()
	assert.NoError(t, err)
}

func TestNewProducer(t *testing.T) {
	p := NewProducer([]string{"localhost:9092"}, "topic")
	assert.NotNil(t, p)
}

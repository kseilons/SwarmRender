package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"kseilons/stitching-service/internall/domain"
)

type MockReader struct {
	mock.Mock
}

func (m *MockReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).(kafka.Message), args.Error(1)
}

func (m *MockReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	args := m.Called(ctx, msgs)
	return args.Error(0)
}

func (m *MockReader) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockReader) Config() kafka.ReaderConfig {
	return kafka.ReaderConfig{Topic: "test-topic"}
}

type MockUseCase struct {
	mock.Mock
}

func (m *MockUseCase) Execute(ctx context.Context, logger *slog.Logger, task domain.StitchingTask) error {
	args := m.Called(ctx, logger, task)
	return args.Error(0)
}

func TestConsumer_Run_ProcessTask(t *testing.T) {
	mockReader := new(MockReader)
	mockUseCase := new(MockUseCase)
	logger := slog.Default()
	c := &Consumer{
		reader:  mockReader,
		useCase: mockUseCase,
		logger:  logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "p1"}}
	payload, _ := json.Marshal(task)
	msg := kafka.Message{Value: payload}

	mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
	mockUseCase.On("Execute", mock.Anything, logger, mock.Anything).Return(nil).Once()
	mockReader.On("CommitMessages", mock.Anything, mock.Anything).Return(nil).Once()

	mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Once()
	cancel()

	err := c.Run(ctx)
	assert.NoError(t, err)
}

func TestConsumer_Run_UnmarshalError(t *testing.T) {
	mockReader := new(MockReader)
	mockUseCase := new(MockUseCase)
	c := &Consumer{
		reader:  mockReader,
		useCase: mockUseCase,
		logger:  slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	msg := kafka.Message{Value: []byte("invalid json")}

	mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
	mockReader.On("CommitMessages", mock.Anything, mock.Anything).Return(nil).Once()
	
	mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Once()
	cancel()

	err := c.Run(ctx)
	assert.NoError(t, err)
}

func TestConsumer_Run_ExecuteError(t *testing.T) {
	mockReader := new(MockReader)
	mockUseCase := new(MockUseCase)
	c := &Consumer{
		reader:  mockReader,
		useCase: mockUseCase,
		logger:  slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "p1"}}
	payload, _ := json.Marshal(task)
	msg := kafka.Message{Value: payload}

	mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
	mockUseCase.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("usecase error")).Once()
	mockReader.On("CommitMessages", mock.Anything, mock.Anything).Return(nil).Once()

	mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Once()
	cancel()

	err := c.Run(ctx)
	assert.NoError(t, err)
}

func TestConsumer_Run_CommitError(t *testing.T) {
	mockReader := new(MockReader)
	mockUseCase := new(MockUseCase)
	c := &Consumer{
		reader:  mockReader,
		useCase: mockUseCase,
		logger:  slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := domain.StitchingTask{BaseTask: domain.BaseTask{ProjectID: "p1"}}
	payload, _ := json.Marshal(task)
	msg := kafka.Message{Value: payload}

	mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
	mockUseCase.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	mockReader.On("CommitMessages", mock.Anything, mock.Anything).Return(errors.New("commit error")).Once()

	mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Once()
	cancel()

	err := c.Run(ctx)
	assert.NoError(t, err)
}

func TestConsumer_Run_FetchError(t *testing.T) {
	mockReader := new(MockReader)
	mockUseCase := new(MockUseCase)
	c := &Consumer{
		reader:  mockReader,
		useCase: mockUseCase,
		logger:  slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, errors.New("kafka error")).Once()
	mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Once()
	cancel()

	err := c.Run(ctx)
	assert.NoError(t, err)
}

func TestConsumer_Close(t *testing.T) {
	mockReader := new(MockReader)
	c := &Consumer{reader: mockReader}
	mockReader.On("Close").Return(nil).Once()
	err := c.Close()
	assert.NoError(t, err)
}

func TestNewConsumer(t *testing.T) {
	c := NewConsumer([]string{"localhost:9092"}, "topic", "group", nil, slog.Default())
	assert.NotNil(t, c)
}

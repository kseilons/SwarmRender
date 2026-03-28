package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/internall/transport/kafka/mocks"
)

func TestConsumer_Run(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Success", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockUC := mocks.NewRenderUseCase(t)
		consumer := &Consumer{
			reader:  mockReader,
			useCase: mockUC,
			logger:  logger,
		}

		task := domain.RenderTask{ProjectID: "test-1"}
		payload, _ := json.Marshal(task)
		msg := kafka.Message{Value: payload, Key: []byte("key")}

		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, errors.New("stop")).Run(func(args mock.Arguments) {
			cancel()
		})
		mockUC.On("Execute", mock.Anything, mock.Anything, task).Return([]byte("ok"), nil).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		
		mockReader.AssertExpectations(t)
		mockUC.AssertExpectations(t)
	})

	t.Run("UnmarshalError", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockUC := mocks.NewRenderUseCase(t)
		consumer := &Consumer{
			reader:  mockReader,
			useCase: mockUC,
			logger:  logger,
		}

		msg := kafka.Message{Value: []byte("invalid json")}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		
		mockReader.AssertExpectations(t)
	})

	t.Run("UseCaseError", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockUC := mocks.NewRenderUseCase(t)
		consumer := &Consumer{
			reader:  mockReader,
			useCase: mockUC,
			logger:  logger,
		}

		task := domain.RenderTask{ProjectID: "test-fail"}
		payload, _ := json.Marshal(task)
		msg := kafka.Message{Value: payload}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockUC.On("Execute", mock.Anything, mock.Anything, task).Return(nil, errors.New("render failed")).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		
		mockReader.AssertExpectations(t)
		mockUC.AssertExpectations(t)
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Once()

		err := consumer.Run(ctx)
		assert.NoError(t, err)
	})
}

func TestConsumer_Close(t *testing.T) {
	mockReader := mocks.NewReader(t)
	consumer := &Consumer{reader: mockReader}

	mockReader.On("Close").Return(nil).Once()
	err := consumer.Close()
	assert.NoError(t, err)
}

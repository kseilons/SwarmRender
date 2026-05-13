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
	"kseilons/worker-service/internall/config"
	"kseilons/worker-service/internall/domain"
	"kseilons/worker-service/internall/transport/kafka/mocks"
)

func TestConsumer_Run(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Success", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockRenderUC := mocks.NewRenderUseCase(t)
		mockMetadataUC := mocks.NewMetadataUseCase(t)
		consumer := &Consumer{
			reader:     mockReader,
			renderUC:   mockRenderUC,
			metadataUC: mockMetadataUC,
			logger:     logger,
		}

		task := domain.RenderTask{
			BaseTask: domain.BaseTask{
				ProjectID: "test-1",
				TaskID:    "task-1",
			},
			MainFileURL: "project.blend",
		}
		payload, _ := json.Marshal(task)
		msg := kafka.Message{
			Value: payload,
			Key:   []byte("key"),
			Topic: "render-tasks",
		}

		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, errors.New("stop")).Run(func(args mock.Arguments) {
			cancel()
		})
		mockRenderUC.On("Execute", mock.Anything, mock.Anything, task).Return([]byte("ok"), nil).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)

		mockReader.AssertExpectations(t)
		mockRenderUC.AssertExpectations(t)
	})

	t.Run("UnmarshalError_Metadata", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		consumer := &Consumer{reader: mockReader, logger: logger}

		msg := kafka.Message{Value: []byte("invalid json"), Topic: "metadata-tasks"}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		mockReader.AssertExpectations(t)
	})

	t.Run("UnmarshalError_Render", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		consumer := &Consumer{reader: mockReader, logger: logger}

		msg := kafka.Message{Value: []byte("invalid json"), Topic: "render-tasks-cpu"}
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
		mockRenderUC := mocks.NewRenderUseCase(t)
		mockMetadataUC := mocks.NewMetadataUseCase(t)
		consumer := &Consumer{
			reader:     mockReader,
			renderUC:   mockRenderUC,
			metadataUC: mockMetadataUC,
			logger:     logger,
		}

		task := domain.RenderTask{
			BaseTask: domain.BaseTask{
				ProjectID: "test-fail",
				TaskID:    "task-fail",
			},
		}
		payload, _ := json.Marshal(task)
		msg := kafka.Message{
			Value: payload,
			Topic: "render-tasks",
		}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockRenderUC.On("Execute", mock.Anything, mock.Anything, task).Return(nil, errors.New("render failed")).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)

		mockReader.AssertExpectations(t)
		mockRenderUC.AssertExpectations(t)
	})

	t.Run("MetadataSuccess", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockRenderUC := mocks.NewRenderUseCase(t)
		mockMetadataUC := mocks.NewMetadataUseCase(t)
		consumer := &Consumer{
			reader:     mockReader,
			renderUC:   mockRenderUC,
			metadataUC: mockMetadataUC,
			logger:     logger,
		}

		task := domain.MetadataTask{
			BaseTask: domain.BaseTask{ProjectID: "p1", TaskID: "t1"},
		}
		payload, _ := json.Marshal(task)
		msg := kafka.Message{Value: payload, Topic: "metadata-tasks"}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockMetadataUC.On("Execute", mock.Anything, mock.Anything, task).Return(nil).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		mockMetadataUC.AssertExpectations(t)
	})

	t.Run("MetadataError", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockMetadataUC := mocks.NewMetadataUseCase(t)
		consumer := &Consumer{
			reader:     mockReader,
			metadataUC: mockMetadataUC,
			logger:     logger,
		}

		task := domain.MetadataTask{
			BaseTask: domain.BaseTask{ProjectID: "p1", TaskID: "t1"},
		}
		payload, _ := json.Marshal(task)
		msg := kafka.Message{Value: payload, Topic: "metadata-tasks"}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockMetadataUC.On("Execute", mock.Anything, mock.Anything, task).Return(errors.New("meta fail")).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		mockMetadataUC.AssertExpectations(t)
	})

	t.Run("UnknownTopic", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		msg := kafka.Message{Value: []byte("data"), Topic: "unknown"}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockReader.On("CommitMessages", mock.Anything, msg).Return(nil).Once()

		_ = consumer.Run(ctx)
		mockReader.AssertExpectations(t)
	})

	t.Run("CommitError", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		mockMetadataUC := mocks.NewMetadataUseCase(t)
		consumer := &Consumer{
			reader:     mockReader,
			metadataUC: mockMetadataUC,
			logger:     logger,
		}

		msg := kafka.Message{Value: []byte("{}"), Topic: "metadata-tasks"}
		ctx, cancel := context.WithCancel(context.Background())

		mockReader.On("FetchMessage", mock.Anything).Return(msg, nil).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})
		mockMetadataUC.On("Execute", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
		mockReader.On("CommitMessages", mock.Anything, msg).Return(errors.New("commit fail")).Once()

		_ = consumer.Run(ctx)
		mockReader.AssertExpectations(t)
	})

	t.Run("FetchError", func(t *testing.T) {
		mockReader := mocks.NewReader(t)
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx, cancel := context.WithCancel(context.Background())

		// First fetch fails, second fetch succeeds to stop the loop
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, errors.New("transient")).Once()
		mockReader.On("FetchMessage", mock.Anything).Return(kafka.Message{}, context.Canceled).Run(func(args mock.Arguments) {
			cancel()
		})

		_ = consumer.Run(ctx)
		mockReader.AssertExpectations(t)
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

func TestNewConsumer(t *testing.T) {
	t.Run("FullConfig", func(t *testing.T) {
		cfg := config.KafkaConfig{
			Brokers:       []string{"localhost:9092"},
			MetadataTopic: "meta",
			RenderTopic:   "render",
			GroupID:       "group",
		}
		logger := slog.Default()
		consumer := NewConsumer(cfg, nil, nil, logger, []string{"CUDA"})
		assert.NotNil(t, consumer)
		_ = consumer.Close()
	})

	t.Run("EmptyMetadataTopic", func(t *testing.T) {
		cfg := config.KafkaConfig{
			Brokers:     []string{"localhost:9092"},
			RenderTopic: "render",
			GroupID:     "group",
		}
		logger := slog.Default()
		consumer := NewConsumer(cfg, nil, nil, logger, nil)
		assert.NotNil(t, consumer)
		_ = consumer.Close()
	})
}

func TestConsumer_Close(t *testing.T) {
	mockReader := mocks.NewReader(t)
	consumer := &Consumer{reader: mockReader}

	mockReader.On("Close").Return(nil).Once()
	err := consumer.Close()
	assert.NoError(t, err)
}

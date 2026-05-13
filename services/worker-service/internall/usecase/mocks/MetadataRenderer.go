package mocks

import (
	"context"
	"kseilons/worker-service/internall/domain"

	"github.com/stretchr/testify/mock"
)

type MetadataRenderer struct {
	mock.Mock
}

func (_m *MetadataRenderer) GetMetadata(ctx context.Context, blendPath string) (domain.ProjectMetadata, error) {
	ret := _m.Called(ctx, blendPath)
	return ret.Get(0).(domain.ProjectMetadata), ret.Error(1)
}

func NewMetadataRenderer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MetadataRenderer {
	mock := &MetadataRenderer{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}

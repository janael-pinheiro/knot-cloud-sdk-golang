package mocks

import (
	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/stretchr/testify/mock"
)

type PublisherMock struct {
	mock.Mock
}

func (p *PublisherMock) PublishDeviceRegister(userToken string, device *entities.Device) error {
	args := p.Called()
	return args.Error(0)
}

func (p *PublisherMock) PublishDeviceUnregister(userToken string, device *entities.Device) error {
	args := p.Called(userToken, device)
	return args.Error(0)
}

func (p *PublisherMock) PublishDeviceAuth(userToken string, device *entities.Device) error {
	args := p.Called()
	return args.Error(0)
}

func (p *PublisherMock) PublishDeviceUpdateConfig(userToken string, device *entities.Device) error {
	args := p.Called()
	return args.Error(0)
}

func (p *PublisherMock) PublishDeviceData(userToken string, device *entities.Device, data []entities.Data) error {
	args := p.Called()
	return args.Error(0)
}

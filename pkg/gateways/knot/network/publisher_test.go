package network

import (
	"errors"
	"testing"

	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/stretchr/testify/assert"
)

func createFakeMessageOptions(userToken, expirationTime string) MessageOptions {
	return MessageOptions{
		Authorization: userToken,
		Expiration:    expirationTime,
	}
}

func createFakeDevice(id, name string) entities.Device {
	return entities.Device{
		ID:   id,
		Name: name,
	}
}

func createFakeDeviceRegisterRequest(id, name string) DeviceRegisterRequest {
	return DeviceRegisterRequest{
		ID:   id,
		Name: name,
	}
}

func TestPublishDeviceRegister(t *testing.T) {
	amqpMock := new(AmqpMock)
	userToken := "token"
	options := createFakeMessageOptions(userToken, defaultExpirationTime)
	device := createFakeDevice("1", "test")
	message := createFakeDeviceRegisterRequest(device.ID, device.Name)

	amqpMock.On("PublishPersistentMessage", exchangeDevice, exchangeTypeDirect, routingKeyRegister, message, &options).Return(nil)

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceRegister(userToken, &device)
	assert.Nil(t, err)
	amqpMock.AssertExpectations(t)
}

func TestPublishDeviceRegisterWhenEmptyTokenReturnError(t *testing.T) {
	amqpMock := new(AmqpMock)
	emptyUserToken := ""
	options := createFakeMessageOptions(emptyUserToken, defaultExpirationTime)
	device := createFakeDevice("1", "test")
	message := createFakeDeviceRegisterRequest(device.ID, device.Name)

	amqpMock.On("PublishPersistentMessage", exchangeDevice, exchangeTypeDirect, routingKeyRegister, message, &options).Return(errors.New("failed"))

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceRegister(emptyUserToken, &device)
	assert.NotNil(t, err)
	amqpMock.AssertExpectations(t)
}

func TestPublishDeviceUnregister(t *testing.T) {
	amqpMock := new(AmqpMock)
	userToken := "token"
	options := createFakeMessageOptions(userToken, defaultExpirationTime)
	device := createFakeDevice("1", "test")
	message := DeviceUnregisterRequest{
		ID: device.ID,
	}

	amqpMock.On("PublishPersistentMessage", exchangeDevice, exchangeTypeDirect, routingKeyUnregister, message, &options).Return(nil)

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceUnregister(userToken, &device)
	assert.Nil(t, err)
	amqpMock.AssertExpectations(t)
}

func TestPublishDeviceUnregisterWhenEmptyTokenReturnError(t *testing.T) {
	amqpMock := new(AmqpMock)
	userToken := ""
	options := createFakeMessageOptions(userToken, defaultExpirationTime)
	device := createFakeDevice("1", "test")
	message := DeviceUnregisterRequest{
		ID: device.ID,
	}

	amqpMock.On("PublishPersistentMessage", exchangeDevice, exchangeTypeDirect, routingKeyUnregister, message, &options).Return(errors.New("failed"))

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceUnregister(userToken, &device)
	assert.NotNil(t, err)
	amqpMock.AssertExpectations(t)
}

func TestPublishDeviceAuth(t *testing.T) {
	amqpMock := new(AmqpMock)
	userToken := "token"
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
		CorrelationID: defaultCorrelationID,
		ReplyTo:       ReplyToAuthMessages,
	}
	device := createFakeDevice("1", "test")
	message := DeviceAuthRequest{
		ID:    device.ID,
		Token: device.Token,
	}

	amqpMock.On("PublishPersistentMessage", exchangeDevice, exchangeTypeDirect, routingKeyAuth, message, &options).Return(nil)

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceAuth(userToken, &device)
	assert.Nil(t, err)
	amqpMock.AssertExpectations(t)
}

func TestPublishDeviceUpdateConfig(t *testing.T) {
	amqpMock := new(AmqpMock)
	userToken := "token"
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
	}
	device := createFakeDevice("1", "test")
	message := ConfigUpdateRequest{
		ID:     device.ID,
		Config: device.Config,
	}

	amqpMock.On("PublishPersistentMessage", exchangeDevice, exchangeTypeDirect, routingKeyUpdateConfig, message, &options).Return(nil)

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceUpdateConfig(userToken, &device)
	assert.Nil(t, err)
	amqpMock.AssertExpectations(t)
}

func TestPublishDeviceData(t *testing.T) {
	amqpMock := new(AmqpMock)
	userToken := "token"
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
	}
	device := createFakeDevice("1", "test")
	data := make([]entities.Data, 1)
	message := DataSent{
		ID:   device.ID,
		Data: data,
	}

	amqpMock.On("PublishPersistentMessage", exchangeSent, exchangeTypeFanout, "", message, &options).Return(nil)

	publisher := NewMsgPublisher(amqpMock)
	err := publisher.PublishDeviceData(userToken, &device, data)
	assert.Nil(t, err)
	amqpMock.AssertExpectations(t)
}

package network

import (
	"fmt"

	"github.com/janael-pinheiro/knot-cloud-sdk-golang/pkg/entities"
)

const (
	routingKeyRegister     = "device.register"
	routingKeyUnregister   = "device.unregister"
	routingKeyAuth         = "device.auth"
	routingKeyUpdateConfig = "device.config.sent"
	defaultCorrelationID   = "default-corrId"
	defaultExpirationTime  = "2000"
)

type Publisher interface {
	PublishDeviceRegister(userToken string, device *entities.Device) error
	PublishDeviceUnregister(userToken string, device *entities.Device) error
	PublishDeviceAuth(userToken string, device *entities.Device) error
	PublishDeviceUpdateConfig(userToken string, device *entities.Device) error
	PublishDeviceData(userToken string, device *entities.Device, data []entities.Data) error
}

type msgPublisher struct {
	amqp Messaging
}

func NewMsgPublisher(amqp Messaging) Publisher {
	return &msgPublisher{amqp}
}

func (mp *msgPublisher) PublishDeviceRegister(userToken string, device *entities.Device) error {
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
	}

	message := DeviceRegisterRequest{
		ID:   device.ID,
		Name: device.Name,
	}

	err := mp.amqp.PublishPersistentMessage(EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, routingKeyRegister, message, &options)
	if err != nil {
		return err
	}

	return nil
}

func (mp *msgPublisher) PublishDeviceUnregister(userToken string, device *entities.Device) error {
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
	}

	message := DeviceUnregisterRequest{
		ID: device.ID,
	}

	err := mp.amqp.PublishPersistentMessage(EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, routingKeyUnregister, message, &options)
	if err != nil {
		return err
	}

	return nil
}

func (mp *msgPublisher) PublishDeviceAuth(userToken string, device *entities.Device) error {
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
		CorrelationID: defaultCorrelationID,
		ReplyTo:       REPLY_TO_AUTH_MESSAGES,
	}

	message := DeviceAuthRequest{
		ID:    device.ID,
		Token: device.Token,
	}

	fmt.Println(message)
	err := mp.amqp.PublishPersistentMessage(EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, routingKeyAuth, message, &options)
	if err != nil {
		return err
	}

	return nil
}

func (mp *msgPublisher) PublishDeviceUpdateConfig(userToken string, device *entities.Device) error {
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
	}

	message := ConfigUpdateRequest{
		ID:     device.ID,
		Config: device.Config,
	}

	err := mp.amqp.PublishPersistentMessage(EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, routingKeyUpdateConfig, message, &options)
	if err != nil {
		return err
	}

	return nil
}

func (mp *msgPublisher) PublishDeviceData(userToken string, device *entities.Device, data []entities.Data) error {
	options := MessageOptions{
		Authorization: userToken,
		Expiration:    defaultExpirationTime,
	}

	message := DataSent{
		ID:   device.ID,
		Data: data,
	}

	err := mp.amqp.PublishPersistentMessage(EXCHANGE_SENT, EXCHANGE_TYPE_FANOUT, "", message, &options)
	if err != nil {
		return err
	}

	return nil
}

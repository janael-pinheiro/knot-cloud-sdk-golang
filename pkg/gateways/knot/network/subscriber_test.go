package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeToKNoTMessages(t *testing.T) {
	amqpMock := new(AmqpMock)
	msgChan := make(chan InMsg)
	amqpMock.On("OnMessage", msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, BindingKeyRegistered).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, BindingKeyUnregistered).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, REPLY_TO_AUTH_MESSAGES).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, BindingKeyUpdatedConfig).Return(nil)
	subscriber := NewMsgSubscriber(amqpMock)
	err := subscriber.SubscribeToKNoTMessages(msgChan)
	assert.NoError(t, err)
	amqpMock.AssertExpectations(t)
}

// msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyRegistered

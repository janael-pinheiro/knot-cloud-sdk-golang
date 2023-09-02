package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeToKNoTMessages(t *testing.T) {
	amqpMock := new(AmqpMock)
	msgChan := make(chan InMsg)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyRegistered).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyUnregistered).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, ReplyToAuthMessages).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyUpdatedConfig).Return(nil)
	subscriber := NewMsgSubscriber(amqpMock)
	err := subscriber.SubscribeToKNoTMessages(msgChan)
	assert.NoError(t, err)
	amqpMock.AssertExpectations(t)
}

// msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyRegistered

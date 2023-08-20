package network

import "testing"

func TestSubscribeToKNoTMessages(t *testing.T) {
	amqpMock := new(AmqpMock)
	msgChan := make(chan InMsg)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyRegistered).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyUnregistered).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, ReplyToAuthMessages).Return(nil)
	amqpMock.On("OnMessage", msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyUpdatedConfig).Return(nil)
	subscriber := NewMsgSubscriber(amqpMock)
	subscriber.SubscribeToKNoTMessages(msgChan)
	amqpMock.AssertExpectations(t)
}

// msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyRegistered

package network

import (
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
)

type AmqpMock struct {
	mock.Mock
}

func (m *AmqpMock) Start() error {
	return nil
}

func (m *AmqpMock) Stop() error { return nil }

func (m *AmqpMock) OnMessage(msgChan chan InMsg, queueName, exchangeName, exchangeType, key string) error {
	args := m.Called(msgChan, queueName, exchangeName, exchangeType, key)
	return args.Error(0)
}

func (m *AmqpMock) PublishPersistentMessage(exchange, exchangeType, key string, data interface{}, options *MessageOptions) error {
	args := m.Called(exchange, exchangeType, key, data, options)
	return args.Error(0)
}

func (m *AmqpMock) Connect() error {
	args := m.Called()
	return args.Error(0)
}

func (m *AmqpMock) DeclareExchange(name, exchangeType string) error {
	args := m.Called(name, exchangeType)
	return args.Error(0)
}

func (m *AmqpMock) DeclareQueue(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *AmqpMock) GetConnection() connection {
	return nil
}

func (m *AmqpMock) GetChannel() *amqp.Channel {
	return nil
}

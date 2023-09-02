package network

import (
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type connectionMock struct {
	mock.Mock
}

func (c *connectionMock) connect() error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) createChannel() error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) getChannel() *amqp.Channel {
	c.Called()
	return nil
}

func (c *connectionMock) queueDeclare(name string) error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) exchangeDeclare(name, exchangeType string) error {
	args := c.Called(name, exchangeType)
	return args.Error(0)
}

func (c *connectionMock) queueBind(queueName, key, exchangeName string, noWait bool, table amqp.Table) error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return nil, nil
}

func (c *connectionMock) publish(exchange string, key string, mandatory bool, immediate bool, data interface{}, options *MessageOptions) error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) isClosed() bool {
	args := c.Called()
	return args.Bool(0)
}

func (c *connectionMock) close() error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) closeChannel() error {
	args := c.Called()
	return args.Error(0)
}

func (c *connectionMock) notifyClose(channel chan *amqp.Error) chan *amqp.Error {
	return nil
}

func TestCreateAMQPConnection(t *testing.T) {
	connectionMock := new(connectionMock)
	connectionMock.On("connect").Return(nil)
	connectionMock.On("createChannel").Return(nil)
	connectionMock.On("exchangeDeclare", "device", "direct").Return(nil)
	connectionMock.On("exchangeDeclare", "data.published", "fanout").Return(nil)
	amqp := NewAMQPHandler(connectionMock)
	err := amqp.Start()
	assert.Nil(t, err)
}

func TestSetupExchanges(t *testing.T) {
	connectionMock := new(connectionMock)
	connectionMock.On("exchangeDeclare", "device", "direct").Return(nil)
	connectionMock.On("exchangeDeclare", "data.published", "fanout").Return(nil)
	amqp := NewAMQPHandler(connectionMock)
	err := setupExchanges(amqp)
	assert.Nil(t, err)
}

func TestCloseAMQPConnection(t *testing.T) {
	connectionMock := new(connectionMock)
	amqp := NewAMQPHandler(connectionMock)
	connectionMock.On("isClosed").Return(false)
	connectionMock.On("close").Return(nil)
	connectionMock.On("closeChannel").Return(nil)
	err := amqp.Stop()
	assert.NoError(t, err)
}

func TestConnect(t *testing.T) {
	connectionMock := new(connectionMock)
	connectionMock.On("connect").Return(nil)
	connectionMock.On("createChannel").Return(nil)
	amqp := NewAMQPHandler(connectionMock)
	err := amqp.Connect()
	assert.Nil(t, err)
}

func TestConnectWhenInvalidURLThenReturnError(t *testing.T) {
	connectionMock := new(connectionMock)
	expectedErrorMessage := "Invalid URL"
	connectionMock.On("connect").Return(errors.New(expectedErrorMessage))
	amqp := NewAMQPHandler(connectionMock)
	err := amqp.Connect()
	assert.NotNil(t, err)
	assert.Equal(t, expectedErrorMessage, err.Error())
}

func TestDeclareQueue(t *testing.T) {
	connectionMock := new(connectionMock)
	connectionMock.On("queueDeclare").Return(nil)
	amqp := NewAMQPHandler(connectionMock)
	testQueueName := "test"
	err := amqp.DeclareQueue(testQueueName)
	assert.Nil(t, err)
}

func TestDeclareExchange(t *testing.T) {
	connectionMock := new(connectionMock)
	testExchangeName := "test"
	testExchangeType := "fanout"
	connectionMock.On("exchangeDeclare", testExchangeName, testExchangeType).Return(nil)
	amqp := NewAMQPHandler(connectionMock)
	err := amqp.DeclareExchange(testExchangeName, testExchangeType)
	assert.Nil(t, err)
}

func TestDeclareExchangeWhenInvalidExchangeTypeReturnError(t *testing.T) {
	connectionMock := new(connectionMock)
	testExchangeName := "test"
	testExchangeType := "invalid"
	expectedErrorMessage := "Invalid exchange type"
	connectionMock.On("exchangeDeclare", testExchangeName, testExchangeType).Return(errors.New(expectedErrorMessage))
	amqp := NewAMQPHandler(connectionMock)
	err := amqp.DeclareExchange(testExchangeName, testExchangeType)
	assert.NotNil(t, err)
	assert.Equal(t, expectedErrorMessage, err.Error())
}

func TestOnMessage(t *testing.T) {
	connectionMock := new(connectionMock)
	amqp := NewAMQPHandler(connectionMock)
	msgChan := make(chan InMsg)
	testExchangeName := "test"
	testExchangeType := "fanout"
	testQueueName := "test"
	testKey := ""
	connectionMock.On("exchangeDeclare", testExchangeName, testExchangeType).Return(nil)
	connectionMock.On("queueDeclare").Return(nil)
	connectionMock.On("queueBind").Return(nil)
	err := amqp.OnMessage(msgChan, testQueueName, testExchangeName, testExchangeType, testKey)
	assert.Nil(t, err)
}

func TestPublishPersistentMessage(t *testing.T) {
	connectionMock := new(connectionMock)
	amqp := NewAMQPHandler(connectionMock)
	testExchangeName := "test"
	testExchangeType := "fanout"
	testKey := ""
	options := MessageOptions{
		Authorization: "",
		Expiration:    "2000",
	}

	message := DeviceRegisterRequest{
		ID:   "",
		Name: "",
	}
	connectionMock.On("exchangeDeclare", testExchangeName, testExchangeType).Return(nil)
	connectionMock.On("publish").Return(nil)
	err := amqp.PublishPersistentMessage(testExchangeName, testExchangeType, testKey, message, &options)
	assert.NoError(t, err)
}

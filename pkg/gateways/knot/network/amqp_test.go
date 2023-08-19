package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAMQPConnection(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
}

func TestCloseAMQPConnection(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
	assert.False(t, amqp.conn.IsClosed())
	amqp.Stop()
	assert.True(t, amqp.conn.IsClosed())
}

func TestConnect(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.connect()
	assert.Nil(t, err)
	assert.NotNil(t, amqp.channel)
}

func TestConnectWhenInvalidURLThenReturnError(t *testing.T) {
	url := ""
	amqp := NewAMQP(url)
	err := amqp.connect()
	assert.NotNil(t, err)
	assert.Nil(t, amqp.channel)
}

func TestDeclareExchange(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
	testExchangeName := "test"
	testExchangeType := "fanout"
	err = amqp.declareExchange(testExchangeName, testExchangeType)
	assert.Nil(t, err)
}

func TestDeclareExchangeWhenInvalidExchangeTypeReturnError(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
	testExchangeName := "test"
	testExchangeType := ""
	err = amqp.declareExchange(testExchangeName, testExchangeType)
	assert.NotNil(t, err)
}

func TestDeclareQueue(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
	testQueueName := "test"
	err = amqp.declareQueue(testQueueName)
	assert.Nil(t, err)
}

func TestOnMessage(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
	msgChan := make(chan InMsg)
	testExchangeName := "test"
	testExchangeType := "fanout"
	testQueueName := "test"
	testKey := ""
	err = amqp.OnMessage(msgChan, testQueueName, testExchangeName, testExchangeType, testKey)
	assert.Nil(t, err)
}

func TestPublishPersistentMessage(t *testing.T) {
	url := "amqp://knot:knot@127.0.0.1:5672"
	amqp := NewAMQP(url)
	err := amqp.Start()
	assert.Nil(t, err)
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
	err = amqp.PublishPersistentMessage(testExchangeName, testExchangeType, testKey, message, &options)
}

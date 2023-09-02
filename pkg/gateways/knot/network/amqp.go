package network

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	amqp "github.com/rabbitmq/amqp091-go"
)

type exchanceProperty struct {
	name         string
	exchangeType string
}

type Messaging interface {
	Start() error
	Stop() error
	OnMessage(msgChan chan InMsg, queueName, exchangeName, exchangeType, key string) error
	PublishPersistentMessage(exchange, exchangeType, key string, data interface{}, options *MessageOptions) error
	Connect() error
	DeclareExchange(name, exchangeType string) error
	DeclareQueue(name string) error
}

const (
	exchangeTypeDirect  = "direct"
	exchangeTypeFanout  = "fanout"
	exchangeDevice      = "device"
	exchangeSent        = "data.sent"
	ReplyToAuthMessages = "sql-auth-rpc"
	durable             = true
	deleteWhenUnused    = false
	exclusive           = false
	noWait              = false
	internal            = false
	noAck               = true
	noLocal             = false
	consumerTag         = ""
)

type AMQPHandler struct {
	connection        connection
	declaredExchanges map[string]struct{}
}

type InMsg struct {
	Exchange      string
	RoutingKey    string
	ReplyTo       string
	CorrelationID string
	Headers       map[string]interface{}
	Body          []byte
}

// MessageOptions represents the message publishing options
type MessageOptions struct {
	Authorization string
	CorrelationID string
	ReplyTo       string
	Expiration    string
}

var exchangeLock *sync.Mutex = &sync.Mutex{}

func NewAMQPHandler(connection connection) Messaging {
	declaredExchanges := make(map[string]struct{})
	return &AMQPHandler{connection, declaredExchanges}
}

func (a *AMQPHandler) Start() error {
	err := backoff.Retry(a.Connect, backoff.NewExponentialBackOff())
	if err != nil {
		return err
	}
	go a.notifyWhenClosed()
	return setupExchanges(a)
}

func setupExchanges(a Messaging) error {
	exchageProperties := make([]exchanceProperty, 2)
	exchageProperties[0] = exchanceProperty{name: "device", exchangeType: "direct"}
	exchageProperties[1] = exchanceProperty{name: "data.published", exchangeType: "fanout"}

	var exchangeErrors []error
	for _, exchange := range exchageProperties {
		err := a.DeclareExchange(exchange.name, exchange.exchangeType)
		exchangeErrors = append(exchangeErrors, err)
	}
	return errors.Join(exchangeErrors...)
}

func (a *AMQPHandler) Stop() error {
	if !a.connection.isClosed() {
		defer a.connection.close()
	}

	return a.connection.closeChannel()
}

func (a *AMQPHandler) OnMessage(msgChan chan InMsg, queueName, exchangeName, exchangeType, key string) error {
	err := a.DeclareExchange(exchangeName, exchangeType)
	if err != nil {
		return err
	}

	err = a.DeclareQueue(queueName)
	if err != nil {
		return err
	}

	err = a.connection.queueBind(
		queueName,
		key,
		exchangeName,
		noWait, // noWait
		nil,    // arguments
	)
	if err != nil {
		return err
	}

	deliveries, err := a.connection.consume(
		queueName,
		consumerTag,
		noAck,
		exclusive,
		noLocal,
		noWait,
		nil, // arguments
	)
	if err != nil {
		return err
	}

	go convertDeliveryToInMsg(deliveries, msgChan)

	return nil
}

func (a *AMQPHandler) PublishPersistentMessage(exchange, exchangeType, key string, data interface{}, options *MessageOptions) error {
	//Reduces communication with the AMQP server by avoiding redeclaring an exchage of the same type.
	err := a.connection.publish(exchange, key, false, false, data, options)
	if err != nil {
		return fmt.Errorf("error publishing message in channel: %w", err)
	}

	return nil
}

func (a *AMQPHandler) notifyWhenClosed() {
	errReason := <-a.connection.notifyClose(make(chan *amqp.Error))
	inicialIntervalSecond := 30 * time.Second
	maxIntervalInMinutes := 5 * time.Minute
	intervalMultiplier := 1.7
	neverStopTryReconnection := time.Duration(0)

	reconnectionBackOff := backoff.NewExponentialBackOff()
	reconnectionBackOff.InitialInterval = inicialIntervalSecond
	reconnectionBackOff.MaxInterval = maxIntervalInMinutes
	reconnectionBackOff.Multiplier = intervalMultiplier
	reconnectionBackOff.MaxElapsedTime = neverStopTryReconnection

	reconnection := func() error {
		err := a.connection.connect()
		if err != nil {
			fmt.Println("Error on Dial func, Cannot connect to KNoT: ", err, "Will retry after", reconnectionBackOff.NextBackOff()/time.Second, "seconds")
			return err
		}

		err = a.connection.createChannel()
		if err != nil {
			fmt.Println("Error to get a channel, Cannot connect to KNoT: ", err, "Will retry after", reconnectionBackOff.NextBackOff()/time.Second, "seconds")
			return err
		}
		fmt.Println("Reconnection to KNoT amqp was successful")

		return nil
	}

	if errReason != nil {
		fmt.Println(errReason)
		err := backoff.Retry(reconnection, reconnectionBackOff)
		if err != nil {
			return
		}
		go a.notifyWhenClosed()
	}
}

func (a *AMQPHandler) Connect() error {
	err := a.connection.connect()
	if err != nil {
		return err
	}

	return a.connection.createChannel()
}

func (a *AMQPHandler) DeclareExchange(name, exchangeType string) error {
	return a.connection.exchangeDeclare(name, exchangeType)
}

func (a *AMQPHandler) DeclareQueue(name string) error {
	return a.connection.queueDeclare(name)
}

func convertDeliveryToInMsg(deliveries <-chan amqp.Delivery, outMsg chan InMsg) {
	for d := range deliveries {
		outMsg <- InMsg{d.Exchange, d.RoutingKey, d.ReplyTo, d.CorrelationId, d.Headers, d.Body}
	}
}

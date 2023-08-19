package network

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/streadway/amqp"
)

const (
	exchangeTypeDirect = "direct"
	exchangeTypeFanout = "fanout"

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

type AMQP struct {
	url               string
	conn              *amqp.Connection
	channel           *amqp.Channel
	queue             *amqp.Queue
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

func NewAMQP(url string) *AMQP {
	declaredExchanges := make(map[string]struct{})
	return &AMQP{url, nil, nil, nil, declaredExchanges}
}

func (a *AMQP) Start() error {
	err := backoff.Retry(a.connect, backoff.NewExponentialBackOff())
	if err != nil {
		return err
	}
	go a.notifyWhenClosed()
	return nil
}

func (a *AMQP) Stop() {
	if a.conn != nil && !a.conn.IsClosed() {
		defer a.conn.Close()
	}

	if a.channel != nil {
		defer a.channel.Close()
	}

}

func (a *AMQP) OnMessage(msgChan chan InMsg, queueName, exchangeName, exchangeType, key string) error {
	err := a.declareExchange(exchangeName, exchangeType)
	if err != nil {
		return err
	}

	err = a.declareQueue(queueName)
	if err != nil {
		return err
	}

	err = a.channel.QueueBind(
		queueName,
		key,
		exchangeName,
		noWait, // noWait
		nil,    // arguments
	)
	if err != nil {
		return err
	}

	deliveries, err := a.channel.Consume(
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

func (a *AMQP) PublishPersistentMessage(exchange, exchangeType, key string, data interface{}, options *MessageOptions) error {
	var headers map[string]interface{}
	var corrID, expTime, replyTo string

	if options != nil {
		headers = map[string]interface{}{
			"Authorization": options.Authorization,
		}
		corrID = options.CorrelationID
		replyTo = options.ReplyTo
		expTime = options.Expiration
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error enconding JSON message: %w", err)
	}

	//Reduces communication with the AMQP server by avoiding redeclaring an exchage of the same type.
	if !a.exchangeAlreadyDeclared(exchange) {
		err = a.declareExchange(exchange, exchangeType)
		if err != nil {
			return fmt.Errorf("error declaring exchange: %w", err)
		} else {
			exchangeLock.Lock()
			a.declaredExchanges[exchange] = struct{}{}
			exchangeLock.Unlock()
		}
	}

	err = a.channel.Publish(
		exchange,
		key,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			Headers:         headers,
			ContentType:     "text/plain",
			ContentEncoding: "",
			DeliveryMode:    amqp.Persistent,
			Priority:        0,
			CorrelationId:   corrID,
			ReplyTo:         replyTo,
			Body:            body,
			Expiration:      expTime,
		},
	)
	if err != nil {
		return fmt.Errorf("error publishing message in channel: %w", err)
	}

	return nil
}

func (a *AMQP) exchangeAlreadyDeclared(exchangeName string) bool {
	exchangeLock.Lock()
	_, ok := a.declaredExchanges[exchangeName]
	exchangeLock.Unlock()
	return ok
}

func (a *AMQP) notifyWhenClosed() {
	errReason := <-a.conn.NotifyClose(make(chan *amqp.Error))

	//randomized interval = RetryInterval * (random value in range [1 - RandomizationFactor, 1 + RandomizationFactor])
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
		conn, err := amqp.Dial(a.url)
		if err != nil {
			fmt.Println("Error on Dial func, Cannot connect to KNoT: ", err, "Will retry after", reconnectionBackOff.NextBackOff()/time.Second, "seconds")
			return err
		}

		a.conn = conn
		channel, err := a.conn.Channel()
		if err != nil {
			fmt.Println("Error to get a channel, Cannot connect to KNoT: ", err, "Will retry after", reconnectionBackOff.NextBackOff()/time.Second, "seconds")
			return err
		}
		fmt.Println("Reconnection to KNoT amqp was successful")
		a.channel = channel

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

func (a *AMQP) connect() error {
	conn, err := amqp.Dial(a.url)
	if err != nil {
		return err
	}

	a.conn = conn
	channel, err := a.conn.Channel()
	if err != nil {
		return err
	}

	a.channel = channel

	return nil
}

func (a *AMQP) declareExchange(name, exchangeType string) error {
	return a.channel.ExchangeDeclare(
		name,
		exchangeType,
		durable,
		deleteWhenUnused,
		internal,
		noWait,
		nil, // arguments
	)
}

func (a *AMQP) declareQueue(name string) error {

	queue, err := a.channel.QueueDeclare(
		name,
		durable,
		deleteWhenUnused,
		exclusive,
		noWait,
		nil, // arguments
	)

	a.queue = &queue
	return err
}

func convertDeliveryToInMsg(deliveries <-chan amqp.Delivery, outMsg chan InMsg) {
	for d := range deliveries {
		outMsg <- InMsg{d.Exchange, d.RoutingKey, d.ReplyTo, d.CorrelationId, d.Headers, d.Body}
	}
}

package network

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type connection interface {
	connect() error
	createChannel() error
	getChannel() *amqp.Channel
	queueDeclare(name string) error
	exchangeDeclare(name, exchangeType string) error
	queueBind(queueName, key, exchangeName string, noWait bool, table amqp.Table) error
	consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	publish(exchange string, key string, mandatory bool, immediate bool, data interface{}, options *MessageOptions) error
	isClosed() bool
	close() error
	closeChannel() error
	notifyClose(channel chan *amqp.Error) chan *amqp.Error
}

type AmqpConnection struct {
	url     string
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   *amqp.Queue
}

func NewAmqpConnection(url string) *AmqpConnection {
	return &AmqpConnection{url: url}
}

func (a *AmqpConnection) connect() error {
	conn, err := amqp.Dial(a.url)
	if err == nil {
		a.conn = conn
	}
	return err
}

func (a *AmqpConnection) createChannel() error {
	channel, err := a.conn.Channel()
	if err == nil {
		a.channel = channel
	}
	return err
}

func (a *AmqpConnection) getChannel() *amqp.Channel {
	return a.channel
}

func (a *AmqpConnection) queueDeclare(name string) error {
	queue, err := a.channel.QueueDeclare(
		name,
		durable,
		deleteWhenUnused,
		exclusive,
		noWait,
		nil, // arguments
	)
	if err == nil {
		a.queue = &queue
	}
	return err

}

func (a *AmqpConnection) exchangeDeclare(name, exchangeType string) error {
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

func (a *AmqpConnection) queueBind(queueName, key, exchangeName string, noWait bool, table amqp.Table) error {
	return a.channel.QueueBind(
		queueName,
		key,
		exchangeName,
		noWait,
		table,
	)
}

func (a *AmqpConnection) consume(queue string, consumer string, autoAck bool, exclusive bool, noLocal bool, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return a.channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func (a *AmqpConnection) publish(exchange string, key string, mandatory bool, immediate bool, data interface{}, options *MessageOptions) error {
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

	return a.channel.Publish(
		exchange,
		key,
		mandatory,
		immediate,
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
}

func (a *AmqpConnection) isClosed() bool {
	return a.conn != nil && !a.conn.IsClosed()
}

func (a *AmqpConnection) close() error {
	return a.conn.Close()
}

func (a *AmqpConnection) closeChannel() error {
	return a.channel.Close()
}

func (a *AmqpConnection) notifyClose(channel chan *amqp.Error) chan *amqp.Error {
	return a.conn.NotifyClose(channel)
}

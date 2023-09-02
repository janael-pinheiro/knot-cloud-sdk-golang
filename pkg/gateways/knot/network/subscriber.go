package network

const (
	queueName               = "sql-knot-messages"
	BindingKeyRegistered    = "device.registered"
	BindingKeyUnregistered  = "device.unregistered"
	BindingKeyUpdatedConfig = "device.config.updated"
)

type Subscriber interface {
	SubscribeToKNoTMessages(msgChan chan InMsg) error
}

type msgSubscriber struct {
	amqp Messaging
}

func NewMsgSubscriber(amqp Messaging) Subscriber {
	return &msgSubscriber{amqp}
}

func (ms *msgSubscriber) SubscribeToKNoTMessages(msgChan chan InMsg) error {
	var err error
	subscribe := func(msgChan chan InMsg, queue, exchange, kind, key string) {
		if err != nil {
			return
		}
		err = ms.amqp.OnMessage(msgChan, queue, exchange, kind, key)
	}

	subscribe(msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, BindingKeyRegistered)
	subscribe(msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, BindingKeyUnregistered)
	subscribe(msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, REPLY_TO_AUTH_MESSAGES)
	subscribe(msgChan, queueName, EXCHANGE_DEVICE, EXCHANGE_TYPE_DIRECT, BindingKeyUpdatedConfig)

	return nil
}

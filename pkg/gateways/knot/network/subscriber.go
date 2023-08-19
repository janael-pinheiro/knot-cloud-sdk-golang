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
	amqp *AMQP
}

func NewMsgSubscriber(amqp *AMQP) Subscriber {
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

	subscribe(msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyRegistered)
	subscribe(msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyUnregistered)
	subscribe(msgChan, queueName, exchangeDevice, exchangeTypeDirect, ReplyToAuthMessages)
	subscribe(msgChan, queueName, exchangeDevice, exchangeTypeDirect, BindingKeyUpdatedConfig)

	return nil
}

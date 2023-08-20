package knot

import (
	"fmt"
	"sync"

	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Integration struct {
	protocol                 Protocol
	sensorIDTimestampMapping map[int]string
	pipeDevices              chan map[string]entities.Device
}

var consumerMutex *sync.Mutex = GetMutex()

var deviceChan = make(chan entities.Device)
var msgChan = make(chan network.InMsg)

func NewKNoTIntegration(pipeDevices chan map[string]entities.Device, conf entities.IntegrationKNoTConfig, log *logrus.Entry, devices map[string]entities.Device) (*Integration, error) {
	var err error
	KNoTInteration := Integration{}
	amqpConnection := network.NewAmqpConnection(conf.URL)
	amqp := network.NewAMQPHandler(amqpConnection)
	err = amqp.Start()
	if err != nil {
		log.Println("KNoT connection error")
	} else {
		log.Println("KNoT connected")
	}
	publisher := network.NewMsgPublisher(amqp)
	subscriber := network.NewMsgSubscriber(amqp)
	fileManagement := new(fileManagement)
	KNoTInteration.protocol, err = newProtocol(pipeDevices, conf, deviceChan, msgChan, log, devices, publisher, subscriber, amqp, fileManagement)
	if err != nil {
		return nil, errors.Wrap(err, "new knot protocol")
	}
	KNoTInteration.sensorIDTimestampMapping = make(map[int]string)
	KNoTInteration.pipeDevices = pipeDevices
	return &KNoTInteration, nil
}

// HandleUplinkEvent sends an UplinkEvent.
func (i *Integration) HandleDevice(device entities.Device) {
	device.State = ""
	deviceChan <- device
}

func (integration *Integration) Close() error {
	return integration.protocol.Close()
}

func (i Integration) Transmit(device entities.Device) {

	var data []entities.Data
	for _, d := range device.Data {
		if !isMeasurementNew(i.sensorIDTimestampMapping, fmt.Sprintf("%v", d.TimeStamp), d.SensorID) {
			continue
		}
		i.sensorIDTimestampMapping = updateSensorIDTimestampMapping(i.sensorIDTimestampMapping, fmt.Sprintf("%v", d.TimeStamp), d.SensorID)
		data = append(data, d)
	}
	if data != nil && device.State == entities.KnotPublishing {
		device.Data = data
		i.HandleDevice(device)
	}
}

func (i Integration) Register(device entities.Device) entities.Device {
	i.HandleDevice(device)
	var d entities.Device
	for devices := range i.pipeDevices {
		consumerMutex.Lock()
		device = i.GetDevice(devices)
		consumerMutex.Unlock()
		if device.State == entities.KnotPublishing {
			d = device
			break
		}
	}
	return d
}

func (i Integration) GetDevice(devices map[string]entities.Device) entities.Device {
	/*
		Returns the first and only device in the mapping.
	*/
	keys := make([]string, 0)
	for key := range devices {
		keys = append(keys, key)
	}
	const firstDeviceIndex = 0
	return devices[keys[firstDeviceIndex]]
}

func isMeasurementNew(tagNameTimestampMapping map[int]string, timestamp string, sensorID int) bool {
	// Checks if the timestamp of the current measurement is different from the previous one.
	// As the database returns the query result temporally ordered,
	// we just need to check if the current timestamp is different from the previous one.
	return tagNameTimestampMapping[sensorID] != timestamp
}

func updateSensorIDTimestampMapping(tagNameTimestampMapping map[int]string, timestamp string, sensorID int) map[int]string {
	consumerMutex.Lock()
	tagNameTimestampMapping[sensorID] = timestamp
	consumerMutex.Unlock()
	return tagNameTimestampMapping
}

func (i Integration) SentDataToKNoT(sensors []entities.Data, device entities.Device) {
	/*
		Structure the data collected from the database in the format expected by KNoT,
		and finally transmits the data to the KNoT Cloud
	*/
	var data []entities.Data
	for _, sensor := range sensors {
		dat := entities.Data{SensorID: sensor.SensorID, TimeStamp: sensor.TimeStamp, Value: sensor.Value}
		data = append(data, dat)
	}
	device.Data = data
	i.Transmit(device)
}

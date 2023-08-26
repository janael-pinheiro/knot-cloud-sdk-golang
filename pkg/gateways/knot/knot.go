package knot

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	bloomFilter "github.com/bits-and-blooms/bloom/v3"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DUPLICATION_FILTER            = "0"
	FILTER_CAPACITY               = "1000000"
	DUPLICATION_PROBABILITY       = "0.01"
	RESET_FILTER_USAGE_PERCENTAGE = "0.75"
)

type Integration struct {
	protocol                        Protocol
	pipeDevices                     chan map[string]entities.Device
	filters                         map[string]*bloomFilter.BloomFilter
	maximumPercentageFilterUsage    float32
	filterCapacity                  uint
	duplicationProbability          float64
	isMeasurementDuplicatedFunction func(string, int, string) bool
}

var consumerMutex *sync.Mutex = GetMutex()
var duplicationMutex *sync.RWMutex = &sync.RWMutex{}

var deviceChan = make(chan entities.Device)
var msgChan = make(chan network.InMsg)

var duplicationFilterFunctionMapping map[string]func(string, int, string) bool = map[string]func(string, int, string) bool{}

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
	KNoTInteration.pipeDevices = pipeDevices
	KNoTInteration.filters = map[string]*bloomFilter.BloomFilter{}
	maximumPercentageFilterUsage, err := strconv.ParseFloat(getValueFromEnvironmentVariable("RESET_FILTER_USAGE_PERCENTAGE", RESET_FILTER_USAGE_PERCENTAGE), 32)
	if err != nil {
		panic("RESET_FILTER_USAGE_PERCENTAGE environment variable with invalid value.")
	}
	KNoTInteration.maximumPercentageFilterUsage = float32(maximumPercentageFilterUsage)
	filterCapacity, capacityErr := strconv.ParseUint(getValueFromEnvironmentVariable("FILTER_CAPACITY", FILTER_CAPACITY), 10, 0)
	duplicationProbability, probabilityErr := strconv.ParseFloat(getValueFromEnvironmentVariable("DUPLICATION_PROBABILITY", DUPLICATION_PROBABILITY), 32)
	if capacityErr != nil || probabilityErr != nil {
		panic("FILTER_CAPACITY and DUPLICATION_PROBABILITY environment variables with invalid values.")
	}
	KNoTInteration.filterCapacity = uint(filterCapacity)
	KNoTInteration.duplicationProbability = float64(duplicationProbability)
	duplicationFilterFunctionMapping[DUPLICATION_FILTER] = func(timestamp string, sensorID int, deviceID string) bool { return false }
	duplicationFilterFunctionMapping["1"] = KNoTInteration.isMeasurementDuplicated
	enableDuplicationFilter := getValueFromEnvironmentVariable("DUPLICATION_FILTER", DUPLICATION_FILTER)
	KNoTInteration.isMeasurementDuplicatedFunction = duplicationFilterFunctionMapping[enableDuplicationFilter]
	return &KNoTInteration, nil
}

// HandleUplinkEvent sends an UplinkEvent.
func (i *Integration) HandleDevice(device entities.Device) {
	if device.State == entities.KnotNew {
		device.State = ""
	}
	deviceChan <- device
}

func (integration *Integration) Close() error {
	return integration.protocol.Close()
}

func (i Integration) Transmit(device entities.Device) {

	var data []entities.Data
	for _, d := range device.Data {
		if i.isMeasurementDuplicatedFunction(fmt.Sprintf("%v", d.TimeStamp), d.SensorID, device.ID) {
			continue
		}
		duplicationMutex.Lock()
		i.filters[device.ID] = i.updateDuplicationFilter(fmt.Sprintf("%v", d.TimeStamp), d.SensorID, device.ID)
		duplicationMutex.Unlock()
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
			duplicationMutex.Lock()
			i.filters[device.ID] = bloomFilter.NewWithEstimates(i.filterCapacity, i.duplicationProbability)
			duplicationMutex.Unlock()
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

func (i Integration) isMeasurementDuplicated(timestamp string, sensorID int, deviceID string) bool {
	// Checks if the timestamp of the current measurement is different from the previous ones.
	return i.filters[deviceID].Test([]byte(fmt.Sprintf("%s_%d", timestamp, sensorID)))
}

func (i Integration) updateDuplicationFilter(timestamp string, sensorID int, deviceID string) *bloomFilter.BloomFilter {
	i.resetDuplicationFilter(deviceID)
	return i.filters[deviceID].Add([]byte(fmt.Sprintf("%s_%d", timestamp, sensorID)))
}

func (i Integration) resetDuplicationFilter(deviceID string) {
	approximatedFilterSize := i.filters[deviceID].ApproximatedSize()
	filterCapacity := i.filters[deviceID].Cap()
	currentPercentageFilterUsage := (float32(approximatedFilterSize) / float32(filterCapacity)) * 100
	if currentPercentageFilterUsage >= i.maximumPercentageFilterUsage {
		i.filters[deviceID].ClearAll()
	}
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

func getValueFromEnvironmentVariable(variableName, defaultValue string) string {
	value := os.Getenv(variableName)
	if value != "" {
		return value
	}
	return defaultValue
}

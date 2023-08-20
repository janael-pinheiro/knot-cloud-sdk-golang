package knot

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Protocol interface provides methods to handle KNoT Protocol
type Protocol interface {
	Close() error
	createDevice(device entities.Device) error
	deleteDevice(id string) error
	updateDevice(device entities.Device) error
	checkData(device entities.Device) error
	checkDeviceConfiguration(device entities.Device) error
	deviceExists(device entities.Device) bool
	generateID(pipeDevices chan map[string]entities.Device, device entities.Device) (string, error)
	checkTimeout(device entities.Device, log *logrus.Entry) entities.Device
	requestsKnot(deviceChan chan entities.Device, device entities.Device, oldState string, curState string, message string, log *logrus.Entry)
	writeDevicesConfigFile(device entities.Device) error
}
type networkWrapper struct {
	amqp       network.Messaging
	publisher  network.Publisher
	subscriber network.Subscriber
}

type protocol struct {
	userToken      string
	network        *networkWrapper
	devices        map[string]entities.Device
	fileManagement filesystemManagement
}

const (
	registerRequestMessage = "send a register request"
	errorTimeoutMessage    = "timeOut"
)

var knotMutex *sync.Mutex = &sync.Mutex{}

func GetMutex() *sync.Mutex {
	return knotMutex
}

type bidingKeyActionMapping map[string]func(network.InMsg, chan entities.Device, *logrus.Entry)

func NewBidingKeyActionMapping() bidingKeyActionMapping {
	newBidingKeyActionMapping := make(bidingKeyActionMapping)
	newBidingKeyActionMapping[network.BindingKeyRegistered] = bindingKeyRegistered
	newBidingKeyActionMapping[network.BindingKeyUnregistered] = bindingKeyUnregistered
	newBidingKeyActionMapping[network.ReplyToAuthMessages] = replyToAuthMessages
	newBidingKeyActionMapping[network.BindingKeyUpdatedConfig] = bindingKeyUpdatedConfig

	return newBidingKeyActionMapping
}

func bindingKeyRegistered(message network.InMsg, deviceChan chan entities.Device, log *logrus.Entry) {
	device := handlerAMQPmessage(message, log)

	if device.Error == "" {
		log.Println("received a registration response with success")
		device.State = entities.KnotRegistered
		deviceChan <- device
		return
	}
	log.Println("received a registration response with a error")
	if device.Error != "thing is already registered" {
		deviceChan <- errorFormat(device, device.Error)
		return
	}
	device.State = entities.KnotAlreadyReg
	deviceChan <- device
}

func bindingKeyUnregistered(message network.InMsg, deviceChan chan entities.Device, log *logrus.Entry) {
	log.Println("received a unregistration response")
	device := handlerAMQPmessage(message, log)
	device.State = entities.KnotForceDelete
	deviceChan <- device
}

func replyToAuthMessages(message network.InMsg, deviceChan chan entities.Device, log *logrus.Entry) {
	device := handlerAMQPmessage(message, log)

	if device.Error != "" {
		// Already registered
		log.Println("received a authentication response with a error")
		device.State = entities.KnotForceDelete
		deviceChan <- device
	} else {
		log.Println("received a authentication response with success")
		device.State = entities.KnotAuth
		deviceChan <- device

	}
}

func bindingKeyUpdatedConfig(message network.InMsg, deviceChan chan entities.Device, log *logrus.Entry) {
	device := handlerAMQPmessage(message, log)

	if device.Error != "" {
		// Already registered
		log.Println("received a config update response with a error")
		deviceChan <- errorFormat(device, device.Error)
	} else {
		log.Println("received a config update response with success")
		device.State = entities.KnotReady
		deviceChan <- device
	}
}

func newProtocol(pipeDevices chan map[string]entities.Device, conf entities.IntegrationKNoTConfig, deviceChan chan entities.Device, msgChan chan network.InMsg, log *logrus.Entry, devices map[string]entities.Device, publisher network.Publisher, subscriber network.Subscriber, amqp network.Messaging, fileManagement filesystemManagement) (Protocol, error) {
	p := &protocol{}

	p.fileManagement = fileManagement
	p.userToken = conf.UserToken
	p.network = new(networkWrapper)
	p.network.amqp = amqp
	p.network.publisher = publisher
	p.network.subscriber = subscriber

	if err := p.network.subscriber.SubscribeToKNoTMessages(msgChan); err != nil {
		log.Errorln("Error to subscribe")
		return p, err
	}
	p.devices = make(map[string]entities.Device)
	p.devices = devices

	newBidingKeyActionMapping := NewBidingKeyActionMapping()
	go handlerKnotAMQP(msgChan, deviceChan, log, newBidingKeyActionMapping)
	go knotStateMachineHandler(pipeDevices, deviceChan, p, log)

	return p, nil
}

// Check for data to be updated
func (p *protocol) checkData(device entities.Device) error {

	if device.Data == nil || len(device.Data) == 0 {
		return errors.New("Invalid data")
	}

	sliceSize := len(device.Data)
	nextData := 0
	for dataIndex, data := range device.Data {
		if data.Value == "" || data.Value == nil {
			return fmt.Errorf("invalid sensor value")
		} else if data.TimeStamp == "" || data.TimeStamp == nil {
			return fmt.Errorf("invalid sensor timestamp")
		}

		nextData = dataIndex + 1
		for nextData < sliceSize {
			if data.SensorID == device.Data[nextData].SensorID {
				return fmt.Errorf("repeated sensor id")
			}
			nextData++
		}
	}

	return nil
}

func isInvalidValueType(valueType int) bool {
	minValueTypeAllow := 1
	maxValueTypeAllow := 5
	return valueType < minValueTypeAllow || valueType > maxValueTypeAllow
}

// Check for device configuration
func (p *protocol) checkDeviceConfiguration(device entities.Device) error {
	sliceSize := len(device.Config)
	nextData := 0

	if device.Config == nil {
		return fmt.Errorf("sensor has no configuration")
	}

	// Check if the ids are correct, no repetition
	for dataIndex, config := range device.Config {
		if isInvalidValueType(config.Schema.ValueType) {
			return fmt.Errorf("invalid sensor id")
		}

		nextData = dataIndex + 1
		for nextData < sliceSize {
			if config.SensorID == device.Config[nextData].SensorID {
				return fmt.Errorf("repeated sensor id")
			}
			nextData++
		}
	}

	return nil

}

// Return all the differences between these two devices
func compareDevices(new, old entities.Device) []string {
	var diff []string
	var differenceMapping map[string]bool = make(map[string]bool)
	differenceMapping["name"] = new.Name != old.Name
	differenceMapping["id"] = new.ID != old.ID
	differenceMapping["size"] = len(new.Config) != len(old.Config)
	differenceMapping["state"] = new.State != old.State
	differenceMapping["token"] = new.Token != old.Token
	differenceMapping["error"] = new.Error != old.Error
	for key, value := range differenceMapping {
		if value {
			diff = append(diff, key)
		}
	}
	if !differenceMapping["size"] {
		for i, config := range new.Config {
			if config.Event != old.Config[i].Event {
				diff = append(diff, "event")
			} else if config.Schema != old.Config[i].Schema {
				diff = append(diff, "schema")
			} else if config.SensorID != old.Config[i].SensorID {
				diff = append(diff, "sendorId")
			} else if config.Schema.TypeID != old.Config[i].Schema.TypeID {
				diff = append(diff, "schemaTypeId")
			} else if config.Schema.Name != old.Config[i].Schema.Name {
				diff = append(diff, "schemaName")
			} else if config.Schema.Unit != old.Config[i].Schema.Unit {
				diff = append(diff, "schemaUnit")
			} else if config.Schema.ValueType != old.Config[i].Schema.ValueType {
				diff = append(diff, "schemaValueType")
			}
		}
	}
	return diff
}

func (p *protocol) updateDevice(received entities.Device) error {
	if _, checkDevice := p.devices[received.ID]; !checkDevice {

		return fmt.Errorf("device do not exist")
	}

	device := p.devices[received.ID]
	copy := device
	justOneDeviceChanges := 1

	if p.checkDeviceConfiguration(received) == nil {
		device.Config = received.Config
	}
	if received.Name != "" {
		device.Name = received.Name
	}
	if received.Token != "" {
		device.Token = received.Token
	}
	if received.Error != "" {
		device.Error = received.Error
	}
	if received.State != "" {
		device.State = received.State
	}
	if p.checkData(received) == nil {
		device.Data = received.Data
	}

	differences := compareDevices(device, copy)
	if len(differences) > justOneDeviceChanges {
		err := p.writeDevicesConfigFile(device)
		if err != nil {
			log.Fatal(err)
		}
	} else if len(differences) == justOneDeviceChanges && differences[0] != "state" {
		// here we verify if there is only one change and it's not a state change
		err := p.writeDevicesConfigFile(device)
		if err != nil {
			log.Fatal(err)
		}
	}
	// update again because the update on func 'writeDevicesConf...' has no data or current state
	p.devices[received.ID] = device

	return nil
}

func (p *protocol) writeDevicesConfigFile(device entities.Device) error {
	device.Data = nil
	device.State = entities.KnotNew
	// update here because we write all devices on the conf file.
	p.devices[device.ID] = device
	data, err := yaml.Marshal(&p.devices)
	if err != nil {
		return err
	}

	err = p.fileManagement.writeDevicesConfigFile(os.Getenv("DEVICE_CONFIG_FILEPATH"), data) //os.WriteFile(os.Getenv("DEVICE_CONFIG_FILEPATH"), data, 0600)
	if err != nil {
		log.Panic(err)
	}

	log.Println("wrote on config file with success")
	return err
}

func (p *protocol) Close() error {
	return p.network.amqp.Stop()
}

func (p *protocol) createDevice(device entities.Device) error {

	if device.State != "" {
		return fmt.Errorf("device cannot be created, unknown source")
	} else {

		device.State = entities.KnotNew

		p.devices[device.ID] = device

		return nil
	}
}

func (p *protocol) generateID(pipeDevices chan map[string]entities.Device, device entities.Device) (string, error) {
	//When a device doesn't was an id, we can't find it to change the id, so we need to delete all files to avoid adding a new device
	//but we don't lose information as the device data is on the device on the function argument
	if !p.deviceExists(device) {
		p.devices = make(map[string]entities.Device)
	} else {
		delete(p.devices, device.ID)
	}
	var err error
	device.ID, err = tokenIDGenerator()
	if err != nil {
		return "", err
	}
	device.Token = ""
	device.State = entities.KnotNew
	device.Data = nil
	p.devices[device.ID] = device

	err = p.writeDevicesConfigFile(device)
	if err != nil {
		return "", err
	}

	log.Print(" generated a new Device ID : ")
	log.Println(device.ID)

	go updateDeviceMap(pipeDevices, p.devices)

	return device.ID, err
}

func (p *protocol) deviceExists(device entities.Device) bool {

	_, checkDevice := p.devices[device.ID]

	return checkDevice
}

func tokenIDGenerator() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

func errorFormat(device entities.Device, strError string) entities.Device {
	device.Error = strError
	device.State = entities.KnotError
	return device
}

func (p *protocol) deleteDevice(id string) error {
	if _, d := p.devices[id]; !d {
		return fmt.Errorf("device do not exist")
	}

	delete(p.devices, id)
	return nil
}

// non-blocking channel to update devices on the other routin
func updateDeviceMap(pipeDevices chan map[string]entities.Device, devices map[string]entities.Device) {
	pipeDevices <- devices
}

func verifyErrors(err error, log *logrus.Entry) {
	if err != nil {
		log.Errorln(err)
	}
}

func initTimeout(deviceChan chan entities.Device, device entities.Device) {
	go func(deviceChan chan entities.Device, device entities.Device) {
		time.Sleep(20 * time.Second)
		device.Error = errorTimeoutMessage
		deviceChan <- device
	}(deviceChan, device)
}

func (p *protocol) requestsKnot(deviceChan chan entities.Device, device entities.Device, oldState string, curState string, message string, log *logrus.Entry) {
	err := p.checkDeviceConfiguration(device)
	if err != nil {
		log.Errorln(err)
	} else {
		device.State = oldState
		initTimeout(deviceChan, device)
		device.State = curState
		knotMutex.Lock()
		err := p.updateDevice(device)
		knotMutex.Unlock()
		if err != nil {
			log.Errorln(err)
		} else {
			states := make(map[string]func(string, *entities.Device) error)
			states[entities.KnotNew] = p.network.publisher.PublishDeviceRegister
			states[entities.KnotRegistered] = p.network.publisher.PublishDeviceAuth
			states[entities.KnotAuth] = p.network.publisher.PublishDeviceUpdateConfig
			if function, ok := states[oldState]; ok {
				err = function(p.userToken, &device)
			}
			verifyErrors(err, log)
			log.Println(message)
		}
	}
}

func knotStateMachineHandler(pipeDevices chan map[string]entities.Device, deviceChan chan entities.Device, p *protocol, log *logrus.Entry) {

	pipeDevices <- p.devices
	for device := range deviceChan {
		if !p.deviceExists(device) {
			if device.ID == "" {
				knotMutex.Lock()
				_, err := p.generateID(pipeDevices, device)
				knotMutex.Unlock()
				if err != nil {
					device.State = entities.KnotOff
					log.Error(err)
				} else {
					log.Println("Generated device id")
				}
			} else if device.Error != errorTimeoutMessage {
				log.Error("device id received does not match the stored")
			}
		} else {
			device = p.checkTimeout(device, log)
			if device.State != entities.KnotOff && device.Error != errorTimeoutMessage {
				knotMutex.Lock()
				err := p.updateDevice(device)
				knotMutex.Unlock()
				verifyErrors(err, log)
				device = p.devices[device.ID]

				if device.Name == "" {
					log.Fatalln("Device has no name")
				} else if device.State == entities.KnotNew && device.Token != "" {
					device.State = entities.KnotRegistered
				}
			} else if device.Error == errorTimeoutMessage {
				device.Error = ""
			}

			switch device.State {

			// If the device status is new, request a device registration
			case entities.KnotNew:

				p.requestsKnot(deviceChan, device, device.State, entities.KnotWaitReg, registerRequestMessage, log)

			// If the device is already registered, ask for device authentication
			case entities.KnotRegistered:

				p.requestsKnot(deviceChan, device, device.State, entities.KnotWaitAuth, "send a auth request", log)

			// The device has a token and authentication was successful.
			case entities.KnotAuth:

				p.requestsKnot(deviceChan, device, device.State, entities.KnotWaitConfig, "send a updateconfig request", log)

			//everything is ok with knot device
			case entities.KnotReady:
				device.State = entities.KnotPublishing
				knotMutex.Lock()
				err := p.updateDevice(device)
				knotMutex.Unlock()
				if err != nil {
					log.Errorln(err)
				} else {
					go updateDeviceMap(pipeDevices, p.devices)
				}
			// Send the new data that comes from the device to Knot Cloud
			case entities.KnotPublishing:
				if p.checkData(device) == nil {
					err := p.network.publisher.PublishDeviceData(p.userToken, &device, device.Data)
					if err != nil {
						log.Errorln(err)
					} else {
						log.Println("Published data")
						device.Data = nil
						knotMutex.Lock()
						err = p.updateDevice(device)
						knotMutex.Unlock()
						verifyErrors(err, log)
					}
				} else {
					log.Println("invalid data, has no data to send")
				}

			// If the device is already registered, ask for device authentication
			case entities.KnotAlreadyReg:

				var err error
				if device.Token == "" {
					knotMutex.Lock()
					device.ID, err = p.generateID(pipeDevices, device)
					knotMutex.Unlock()
					if err != nil {
						log.Error(err)
					} else {
						p.requestsKnot(deviceChan, device, entities.KnotNew, entities.KnotWaitReg, registerRequestMessage, log)
					}
				} else {

					p.requestsKnot(deviceChan, device, entities.KnotRegistered, entities.KnotWaitAuth, "send a Auth request", log)

				}

			// Just delete
			case entities.KnotForceDelete:
				var err error
				log.Println("delete a device")
				knotMutex.Lock()
				device.ID, err = p.generateID(pipeDevices, device)
				knotMutex.Unlock()
				if err != nil {
					log.Error(err)
				} else {
					p.requestsKnot(deviceChan, device, entities.KnotNew, entities.KnotWaitReg, registerRequestMessage, log)
				}

			// Handle errors
			case entities.KnotError:
				log.Println("ERROR: ")
				switch device.Error {
				case "thing's config not provided":
					log.Println("thing's config not provided")

				default:
					log.Println("ERROR WITHOUT HANDLER" + device.Error)

				}
				device.State = entities.KnotNew
				device.Error = ""
				knotMutex.Lock()
				err := p.updateDevice(device)
				knotMutex.Unlock()
				verifyErrors(err, log)

			// ignore the device
			case entities.KnotOff:

			}
		}
	}
}

func (p *protocol) checkTimeout(device entities.Device, log *logrus.Entry) entities.Device {
	if device.Error == errorTimeoutMessage {
		stateMapping := map[string]string{
			entities.KnotNew:        entities.KnotWaitReg,
			entities.KnotRegistered: entities.KnotWaitAuth,
			entities.KnotAuth:       entities.KnotWaitConfig,
		}
		curDevice := p.devices[device.ID]
		if _, ok := stateMapping[device.State]; ok {
			if stateMapping[device.State] == curDevice.State {
				log.Println(fmt.Sprintf("error: %s", errorTimeoutMessage))
				return device
			}
		}
		device.State = entities.KnotOff
	}
	return device
}

func handlerAMQPmessage(message network.InMsg, log *logrus.Entry) entities.Device {
	receiver := network.DeviceGenericMessage{}
	device := entities.Device{}
	err := json.Unmarshal([]byte(string(message.Body)), &receiver)
	verifyErrors(err, log)
	device.ID = receiver.ID
	device.Name = receiver.Name
	device.Error = receiver.Error
	if network.BindingKeyRegistered == message.RoutingKey && receiver.Token != "" {
		device.Token = receiver.Token
	}
	return device
}

func handlerKnotAMQP(msgChan <-chan network.InMsg, deviceChan chan entities.Device, log *logrus.Entry, bidingKeysActionMapping bidingKeyActionMapping) {
	for message := range msgChan {
		if function, ok := bidingKeysActionMapping[message.RoutingKey]; ok {
			function(message, deviceChan, log)
		}
	}
}

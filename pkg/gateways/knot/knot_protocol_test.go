package knot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network/mocks"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/logging"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const deviceID = "0d7cd9d221385e1a"

func loadConfiguration() (map[string]entities.Device, entities.IntegrationKNoTConfig) {
	var deviceConfiguration map[string]entities.Device = make(map[string]entities.Device)
	var schema entities.Schema = entities.Schema{ValueType: 2, Unit: 1, TypeID: 65296, Name: "dcsn_cntry"}
	var event entities.Event = entities.Event{Change: true, TimeSec: 30, LowerThreshold: nil, UpperThreshold: nil}
	var config entities.Config = entities.Config{SensorID: 1, Schema: schema, Event: event}
	device := new(entities.Device)
	device.ID = deviceID
	device.Token = "398b9a7b-c9d7-4290-be33-d4857b50e9e9"
	device.Name = "test1"
	device.Config = append(device.Config, config)
	device.State = entities.KnotNew
	device.Data = make([]entities.Data, 1)
	device.Error = ""
	deviceConfiguration[deviceID] = *device

	knotConfiguration := entities.IntegrationKNoTConfig{
		URL:       "amqp://knot:knot@192.168.1.9:5672",
		UserToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjI4Nzg0MzEsImp0aSI6Ijc4MDhhYzBkLTRhMDEtNGRkNS04ZmU5LTUyODIzODFmYmMzYSIsImlhdCI6MTY5MTM0MjQzMSwiaXNzIjoicmVzdEBnbWFpbC5jb20iLCJ0eXBlIjoyfQ.rFsPdLNdUreJwVz49QZDWZRg53RqndrHijZ9iVfS1t4"}

	return deviceConfiguration, knotConfiguration
}

func setUp() entities.Device {
	knotDevice := entities.Device{}
	knotDevice.ID = "1"
	return knotDevice
}

func createData(sensorID int) entities.Data {
	return entities.Data{
		SensorID:  sensorID,
		Value:     0,
		TimeStamp: "2022-01-02 07:45:04",
	}

}

func createDataWithEmptyTimestamp(sensorID int) entities.Data {
	return entities.Data{
		SensorID:  sensorID,
		Value:     0,
		TimeStamp: nil,
	}

}

func createDataWithInvalidValue(sensorID int) entities.Data {
	return entities.Data{
		SensorID:  sensorID,
		Value:     nil,
		TimeStamp: "2006-01-01T21:01:25.0-0300",
	}

}

func createConfig(sensorID int) entities.Config {
	return entities.Config{
		SensorID: sensorID,
		Schema:   entities.Schema{ValueType: 1, Unit: 0, TypeID: 65296},
		Event:    entities.Event{Change: true, TimeSec: 30},
	}

}

type newProtocolSuite struct {
	suite.Suite
	err        error
	protocol   protocol
	fakeDevice entities.Device
}

func (protocolSuite *newProtocolSuite) SetupTest() {
	protocolSuite.fakeDevice = setUp()
	protocolSuite.protocol = protocol{
		devices: make(map[string]entities.Device),
	}
	protocolSuite.err = protocolSuite.protocol.createDevice(protocolSuite.fakeDevice)
	assert.NoError(protocolSuite.T(), protocolSuite.err)
}

func (protocolSuite *newProtocolSuite) TestGivenEmptyStateThenCreateDeviceWithKnotNewState() {
	assert.Equal(protocolSuite.T(), protocolSuite.protocol.devices[protocolSuite.fakeDevice.ID].State, entities.KnotNew)
}

func (protocolSuite *newProtocolSuite) TestGivenNonEmptyStateThenFailToCreateDevice() {
	protocolSuite.fakeDevice.State = "invalid"
	err := protocolSuite.protocol.createDevice(protocolSuite.fakeDevice)
	assert.Error(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenExistingDeviceThenTrue() {
	protocolSuite.fakeDevice.State = ""
	protocolSuite.protocol.devices[protocolSuite.fakeDevice.ID] = protocolSuite.fakeDevice
	foundDevice := protocolSuite.protocol.deviceExists(protocolSuite.protocol.devices[protocolSuite.fakeDevice.ID])
	assert.Equal(protocolSuite.T(), foundDevice, true)
}

func (protocolSuite *newProtocolSuite) TestGivenNotExistingDeviceThenFalse() {
	const invalidID = "42"
	foundDevice := protocolSuite.protocol.deviceExists(protocolSuite.protocol.devices[invalidID])
	assert.Equal(protocolSuite.T(), foundDevice, false)
}

func (protocolSuite *newProtocolSuite) TestGivenValidDataThenSucess() {
	var fakeData []entities.Data
	for i := 1; i <= 10; i++ {
		fakeData = append(fakeData, createData(i))
	}
	protocolSuite.fakeDevice.Data = fakeData
	err := protocolSuite.protocol.checkData(protocolSuite.fakeDevice)
	assert.NoError(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenInvalidDataThenError() {
	/*
		invalid starts with sensorID as 0. Why? I do not know.
	*/
	var fakeData []entities.Data
	for i := 0; i < 2; i++ {
		fakeData = append(fakeData, createData(1))
	}
	protocolSuite.fakeDevice.Data = fakeData
	err := protocolSuite.protocol.checkData(protocolSuite.fakeDevice)
	assert.Error(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenInvalidTimestampDataThenError() {
	var fakeData []entities.Data
	for i := 1; i <= 10; i++ {
		fakeData = append(fakeData, createDataWithEmptyTimestamp(i))
	}
	protocolSuite.fakeDevice.Data = fakeData
	err := protocolSuite.protocol.checkData(protocolSuite.fakeDevice)
	assert.Error(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenInvalidValueDataThenError() {
	var fakeData []entities.Data
	for i := 1; i <= 10; i++ {
		fakeData = append(fakeData, createDataWithInvalidValue(i))
	}
	protocolSuite.fakeDevice.Data = fakeData
	err := protocolSuite.protocol.checkData(protocolSuite.fakeDevice)
	assert.Error(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenValidConfigThenSucess() {
	var fakeConfig []entities.Config
	for i := 1; i <= 10; i++ {
		fakeConfig = append(fakeConfig, createConfig(i))
	}
	protocolSuite.fakeDevice.Config = fakeConfig

	err := protocolSuite.protocol.checkDeviceConfiguration(protocolSuite.fakeDevice)
	assert.NoError(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenInvalidConfigThenError() {
	var fakeConfig []entities.Config
	repeatedIdentifier := 0
	for i := 0; i < 10; i++ {
		fakeConfig = append(fakeConfig, createConfig(repeatedIdentifier))
	}
	protocolSuite.fakeDevice.Config = fakeConfig
	err := protocolSuite.protocol.checkDeviceConfiguration(protocolSuite.fakeDevice)
	assert.Error(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenInvalidValueTypeConfigThenError() {
	var fakeConfig []entities.Config
	for i := 0; i < 10; i++ {
		config := createConfig(i)
		config.Schema.ValueType = 10
		fakeConfig = append(fakeConfig, config)
	}
	protocolSuite.fakeDevice.Config = fakeConfig
	err := protocolSuite.protocol.checkDeviceConfiguration(protocolSuite.fakeDevice)
	assert.ErrorContains(protocolSuite.T(), err, "invalid sensor id")
}

func (protocolSuite *newProtocolSuite) TestGivenNoConfigThenError() {
	protocolSuite.fakeDevice.Config = nil
	err := protocolSuite.protocol.checkDeviceConfiguration(protocolSuite.fakeDevice)
	assert.Error(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenValidDeviceIDThenDeleteIt() {
	err := protocolSuite.protocol.deleteDevice(protocolSuite.fakeDevice.ID)
	assert.NoError(protocolSuite.T(), err)
}

func (protocolSuite *newProtocolSuite) TestGivenInvalidDeviceIDThenError() {
	err := protocolSuite.protocol.deleteDevice("fakeID")
	assert.Error(protocolSuite.T(), err)
}

func TestNewProtocolSuite(t *testing.T) {
	suite.Run(t, new(newProtocolSuite))
}

func TestTokenIDGenerator(t *testing.T) {

	_, err := tokenIDGenerator()

	assert.NoError(t, err)
}

func sendDeviceToChannel(device entities.Device, channel chan entities.Device) {
	channel <- device
}

func createChannels() (chan map[string]entities.Device, chan entities.Device) {
	pipeDevices := make(chan map[string]entities.Device)
	deviceChannel := make(chan entities.Device)

	return pipeDevices, deviceChannel
}

func createLogger() *logrus.Entry {
	logrus := logging.NewLogrus("info", os.Stdout)
	logger := logrus.Get("Main")
	return logger
}

func createNullLogger() (*logrus.Entry, *test.Hook) {
	log, hook := test.NewNullLogger()
	logger := log.WithFields(logrus.Fields{
		"Context": "testing",
	})
	return logger, hook
}

func TestGivenInvalidConfigTimeout(t *testing.T) {
	conf := entities.IntegrationKNoTConfig{
		UserToken:               "",
		URL:                     "",
		EventRoutingKeyTemplate: "",
	}
	devices := make(map[string]entities.Device)
	pipeDevices, deviceChan := createChannels()
	logger := createLogger()
	msgChan := make(chan network.InMsg)
	var err error
	os.Setenv("DEVICE_CONFIG_FILEPATH", "fake_device_configuration_filepath.yaml")
	go func() {
		publisherMock := new(mocks.PublisherMock)
		subscriberMock := new(mocks.SubscriberMock)
		amqpMock := new(network.AmqpMock)
		subscriberMock.On("SubscribeToKNoTMessages", msgChan).Return(nil)
		fileManagementMock := new(fileManagementMock)
		fileManagementMock.On("writeDevicesConfigFile").Return(nil)
		_, err = newProtocol(pipeDevices, conf, deviceChan, msgChan, logger, devices, publisherMock, subscriberMock, amqpMock, fileManagementMock)
	}()

	timeout := time.Second * 30
	fmt.Println("Please, wait 60 seconds!")
	time.Sleep(timeout)

	if err == nil {
		err = fmt.Errorf("Timeout")
	}
	assert.Error(t, err)
}

type deviceTimeoutSuite struct {
	suite.Suite
	device              entities.Device
	protocol            Protocol
	logger              *logrus.Entry
	hook                *test.Hook
	deviceConfiguration map[string]entities.Device
}

func (deviceSuite *deviceTimeoutSuite) SetupTest() {
	deviceConfiguration, knotConfiguration := loadConfiguration()
	deviceSuite.deviceConfiguration = deviceConfiguration
	pipeDevices, deviceChan := createChannels()
	deviceSuite.logger, deviceSuite.hook = createNullLogger()
	msgChan := make(chan network.InMsg)
	publisherMock := new(mocks.PublisherMock)
	subscriberMock := new(mocks.SubscriberMock)
	amqpMock := new(network.AmqpMock)
	subscriberMock.On("SubscribeToKNoTMessages", msgChan).Return(nil)
	fileManagementMock := new(fileManagementMock)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	deviceSuite.protocol, _ = newProtocol(pipeDevices, knotConfiguration, deviceChan, msgChan, deviceSuite.logger, deviceConfiguration, publisherMock, subscriberMock, amqpMock, fileManagementMock)
	deviceSuite.device = deviceConfiguration["2801d924dcf1fc52"]
	deviceSuite.device.Error = errorTimeoutMessage
}

func (deviceSuite *deviceTimeoutSuite) TestGivenDeviceWithoutTimeoutCheckTimeout() {
	deviceSuite.device.Error = ""
	deviceWithCheckedTimeout := deviceSuite.protocol.checkTimeout(deviceSuite.device, deviceSuite.logger)
	assert.Equal(deviceSuite.T(), deviceWithCheckedTimeout.Error, deviceSuite.device.Error)
}

func (deviceSuite *deviceTimeoutSuite) TestGivenDeviceTimeoutCheckTimeoutStateNewWaitReg() {
	oldDevice := deviceSuite.device
	oldDevice.State = entities.KnotWaitReg
	deviceSuite.deviceConfiguration[oldDevice.ID] = oldDevice
	deviceSuite.device.State = entities.KnotNew
	_ = deviceSuite.protocol.checkTimeout(deviceSuite.device, deviceSuite.logger)
	expectedErrorMessage := "error: timeOut"
	assert.Equal(deviceSuite.T(), deviceSuite.hook.LastEntry().Message, expectedErrorMessage)
}

func (deviceSuite *deviceTimeoutSuite) TestGivenDeviceTimeoutCheckTimeoutStateRegisteredWaitAuth() {
	oldDevice := deviceSuite.device
	oldDevice.State = entities.KnotWaitAuth
	deviceSuite.deviceConfiguration[oldDevice.ID] = oldDevice
	deviceSuite.device.State = entities.KnotRegistered

	_ = deviceSuite.protocol.checkTimeout(deviceSuite.device, deviceSuite.logger)
	expectedErrorMessage := "error: timeOut"
	assert.Equal(deviceSuite.T(), deviceSuite.hook.LastEntry().Message, expectedErrorMessage)
}

func (deviceSuite *deviceTimeoutSuite) TestGivenDeviceTimeoutCheckTimeoutStateAuthWaitConfig() {
	oldDevice := deviceSuite.device
	oldDevice.State = entities.KnotWaitConfig
	deviceSuite.deviceConfiguration[oldDevice.ID] = oldDevice
	deviceSuite.device.State = entities.KnotAuth
	_ = deviceSuite.protocol.checkTimeout(deviceSuite.device, deviceSuite.logger)
	expectedErrorMessage := "error: timeOut"
	assert.Equal(deviceSuite.T(), deviceSuite.hook.LastEntry().Message, expectedErrorMessage)
}

func (deviceSuite *deviceTimeoutSuite) TestGivenDeviceTimeoutCheckTimeoutStateNew() {
	deviceWithCheckedTimeout := deviceSuite.protocol.checkTimeout(deviceSuite.device, deviceSuite.logger)
	assert.Equal(deviceSuite.T(), deviceWithCheckedTimeout.State, entities.KnotOff)
}

func TestDeviceTimeoutSuite(t *testing.T) {
	suite.Run(t, new(deviceTimeoutSuite))
}

type testCase struct {
	bidingKey       string
	expectedMessage string
	testName        string
	errorMessage    string
}

var testCases = []testCase{
	{bidingKey: network.BindingKeyRegistered, expectedMessage: "received a registration response with success", testName: "TestGivenBindingKeyRegisteredDeviceWithoutErrorHandleKnotAMQP"},
	{bidingKey: network.BindingKeyRegistered, expectedMessage: "received a registration response with a error", testName: "TestGivenBindingKeyRegisteredDeviceWithErrorHandleKnotAMQP", errorMessage: "Testing error"},
	{bidingKey: network.BindingKeyUpdatedConfig, expectedMessage: "received a config update response with a error", testName: "TestGivenBindingKeyUpdatedConfigWithoutErrorKnotAMQP", errorMessage: "testing error"},
}

func TestHandleKnotAMQP(t *testing.T) {

	for _, test := range testCases {
		logger, hook := createNullLogger()
		deviceChan := make(chan entities.Device)
		msgChan := make(chan network.InMsg)
		testAMQPMessage := network.InMsg{
			RoutingKey: test.bidingKey,
		}
		testDevice := entities.Device{ID: "42", Error: test.errorMessage}
		testAMQPMessage.Body, _ = json.Marshal(testDevice)
		newBidingKeyActionMapping := NewBidingKeyActionMapping()
		go handlerKnotAMQP(msgChan, deviceChan, logger, newBidingKeyActionMapping)
		msgChan <- testAMQPMessage
		<-deviceChan
		assert.Equal(t, hook.LastEntry().Message, test.expectedMessage, test.testName)
	}

}

type testCaseWithState struct {
	bidingKey       string
	expectedMessage string
	testName        string
	errorMessage    string
	state           string
}

var testCasesWithState = []testCaseWithState{
	{bidingKey: network.BindingKeyRegistered, expectedMessage: "received a registration response with a error", testName: "TestGivenBindingKeyRegisteredDeviceAlreadyRegisteredHandleKnotAMQP", errorMessage: "thing is already registered", state: entities.KnotAlreadyReg},
	{bidingKey: network.BindingKeyUnregistered, expectedMessage: "received a unregistration response", testName: "TestGivenBindingKeyUnregisteredHandleKnotAMQP", errorMessage: "", state: entities.KnotForceDelete},
	{bidingKey: network.ReplyToAuthMessages, expectedMessage: "received a authentication response with success", testName: "TestGivenReplyToAuthMessagesHandleWithoutErrorKnotAMQP", errorMessage: "", state: entities.KnotAuth},
	{bidingKey: network.ReplyToAuthMessages, expectedMessage: "received a authentication response with a error", testName: "TestGivenReplyToAuthMessagesHandleWithoutErrorKnotAMQP", errorMessage: "testing error", state: entities.KnotForceDelete},
	{bidingKey: network.BindingKeyUpdatedConfig, expectedMessage: "received a config update response with success", testName: "TestGivenBindingKeyUpdatedConfigWithoutErrorKnotAMQP", errorMessage: "", state: entities.KnotReady},
}

func TestGivenBindingKeyRegisteredDeviceAlreadyRegisteredHandleKnotAMQP(t *testing.T) {

	for _, test := range testCasesWithState {
		logger, hook := createNullLogger()
		deviceChan := make(chan entities.Device)
		msgChan := make(chan network.InMsg)
		testAMQPMessage := network.InMsg{
			RoutingKey: test.bidingKey,
		}
		testDevice := entities.Device{ID: "42", Error: test.errorMessage}
		testAMQPMessage.Body, _ = json.Marshal(testDevice)
		newBidingKeyActionMapping := NewBidingKeyActionMapping()
		go handlerKnotAMQP(msgChan, deviceChan, logger, newBidingKeyActionMapping)
		msgChan <- testAMQPMessage
		device := <-deviceChan
		assert.Equal(t, device.Error, test.errorMessage)
		assert.Equal(t, hook.LastEntry().Message, test.expectedMessage)
		assert.Equal(t, device.State, test.state)
	}
}

func TestCompareDevices(t *testing.T) {
	var fakeConfig1 []entities.Config
	for i := 1; i <= 1; i++ {
		fakeConfig1 = append(fakeConfig1, createConfig(i))
	}
	var fakeConfig2 []entities.Config
	for i := 1; i <= 1; i++ {
		fakeConfig2 = append(fakeConfig2, createConfig(i))
	}
	testDevice1 := entities.Device{ID: "42", Name: "testDevice1", State: entities.KnotAlreadyReg, Token: "token1", Error: "error1", Config: fakeConfig1}
	testDevice2 := entities.Device{ID: "73", Name: "testDevice2", State: entities.KnotNew, Token: "token2", Error: "error2", Config: fakeConfig2}
	var expectedDifferences []string = []string{"name", "id", "state", "token", "error"}
	differences := compareDevices(testDevice1, testDevice2)
	assert.ElementsMatch(t, expectedDifferences, differences)
}

func TestUpdateDevice(t *testing.T) {
	fakeDevice := setUp()
	fileManagementMock := new(fileManagementMock)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	newProtocol := protocol{
		devices:        make(map[string]entities.Device),
		fileManagement: fileManagementMock,
	}
	err := newProtocol.createDevice(fakeDevice)

	assert.NoError(t, err)
	err = newProtocol.updateDevice(fakeDevice)
	assert.NoError(t, err)
}

func TestUpdateDeviceWhenNonExistentDeviceThenReturnError(t *testing.T) {
	fakeDevice := setUp()
	fileManagementMock := new(fileManagementMock)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	newProtocol := protocol{
		devices:        make(map[string]entities.Device),
		fileManagement: fileManagementMock,
	}
	err := newProtocol.createDevice(fakeDevice)

	assert.NoError(t, err)
	fakeDevice.ID = "2"
	err = newProtocol.updateDevice(fakeDevice)
	assert.Error(t, err)
	assert.Equal(t, "device do not exist", err.Error())
}

func TestGenerateID(t *testing.T) {
	fakeDevice := setUp()
	fileManagementMock := new(fileManagementMock)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	newProtocol := protocol{
		devices:        make(map[string]entities.Device),
		fileManagement: fileManagementMock,
	}
	pipeDevices, _ := createChannels()
	deviceID, err := newProtocol.generateID(pipeDevices, fakeDevice)
	assert.NoError(t, err)
	assert.Equal(t, newProtocol.devices[deviceID].State, entities.KnotNew)
}

type testCaseRequestsKnot struct {
	oldState             string
	currentState         string
	expectedMessage      string
	expectedErrorMessage string
	functionName         string
}

var testCasesRequestsKnot = []testCaseRequestsKnot{
	{entities.KnotNew, entities.KnotWaitConfig, "send a register request", "registerError", "PublishDeviceRegister"},
	{entities.KnotRegistered, entities.KnotWaitAuth, "send a auth request", "authError", "PublishDeviceAuth"},
	{entities.KnotAuth, entities.KnotWaitConfig, "send a updateconfig request", "updateConfigError", "PublishDeviceUpdateConfig"},
}

var testCasesRequestsKnotStates = []testCaseRequestsKnot{
	{entities.KnotNew, entities.KnotWaitConfig, "send a register request", "registerError", "PublishDeviceRegister"},
	{entities.KnotRegistered, entities.KnotWaitAuth, "send a auth request", "authError", "PublishDeviceAuth"},
	{entities.KnotAuth, entities.KnotWaitConfig, "send a updateconfig request", "updateConfigError", "PublishDeviceUpdateConfig"},
	{entities.KnotAlreadyReg, "", "send a register request", "knotAlreadyRegError", "PublishDeviceRegister"},
}

type testState struct {
	currentState         string
	expectedErrorMessage string
	deviceData           []entities.Data
	expectedError        error
}

var testCasesKnotStateMachineHandlerStates = []testState{
	{entities.KnotPublishing, "invalid data, has no data to send", append([]entities.Data{}, createDataWithEmptyTimestamp(1)), nil},
	{entities.KnotPublishing, "Published data", append([]entities.Data{}, createData(1)), nil},
}

type requestKNotSuite struct {
	suite.Suite
	device        entities.Device
	protocol      Protocol
	logger        *logrus.Entry
	hook          *test.Hook
	publisherMock *mocks.PublisherMock
	pipeDevices   chan map[string]entities.Device
}

func (knot *requestKNotSuite) SetupTest() {
	os.Setenv("DEVICE_CONFIG_FILEPATH", "/tmp/device_config.yaml")
	deviceConfiguration, knotConfiguration := loadConfiguration()
	knot.pipeDevices, deviceChan = createChannels()
	knot.logger, knot.hook = createNullLogger()
	msgChan := make(chan network.InMsg)
	knot.publisherMock = new(mocks.PublisherMock)
	subscriberMock := new(mocks.SubscriberMock)
	amqpMock := new(network.AmqpMock)
	fileManagementMock := new(fileManagementMock)
	subscriberMock.On("SubscribeToKNoTMessages", msgChan).Return(nil)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	knot.protocol, _ = newProtocol(knot.pipeDevices, knotConfiguration, deviceChan, msgChan, knot.logger, deviceConfiguration, knot.publisherMock, subscriberMock, amqpMock, fileManagementMock)
	knot.device = deviceConfiguration[deviceID]
	var fakeConfig []entities.Config
	for i := 1; i <= 10; i++ {
		fakeConfig = append(fakeConfig, createConfig(i))
	}
	knot.device.Config = fakeConfig
}

func (knot *requestKNotSuite) TestRequestsKnot() {
	for _, test := range testCasesRequestsKnot {
		knot.publisherMock.On(test.functionName).Return(nil)
		knot.protocol.requestsKnot(deviceChan, knot.device, test.oldState, test.currentState, test.expectedMessage, knot.logger)
		assert.Equal(knot.T(), test.expectedMessage, knot.hook.LastEntry().Message)
	}
}

func (knot *requestKNotSuite) TestRequestsKnotWhenKNoTCommucationFailReturnError() {
	for _, test := range testCasesRequestsKnot {
		knot.publisherMock.On(test.functionName).Return(errors.New(test.expectedErrorMessage))
		knot.protocol.requestsKnot(deviceChan, knot.device, test.oldState, test.currentState, test.expectedMessage, knot.logger)
		var logMessages []string
		for _, message := range knot.hook.Entries {
			logMessages = append(logMessages, message.Message)
		}
		assert.Contains(knot.T(), logMessages, test.expectedErrorMessage)
	}
}

func (knot *requestKNotSuite) TestKnotStateMachineHandler() {
	<-knot.pipeDevices
	for _, test := range testCasesKnotStateMachineHandlerStates {
		knot.publisherMock.On("PublishDeviceData").Return(test.expectedError)
		knot.device.State = test.currentState
		knot.device.Data = test.deviceData
		deviceChan <- knot.device
		time.Sleep(1 * time.Second)
		var logMessages []string
		for _, message := range knot.hook.Entries {
			logMessages = append(logMessages, message.Message)
		}
		assert.Contains(knot.T(), logMessages, test.expectedErrorMessage)
	}
}

func (knot *requestKNotSuite) TestKnotStateMachineHandlerWhenCommunicationProblemThenError() {
	var fakeData []entities.Data
	for i := 1; i <= 10; i++ {
		fakeData = append(fakeData, createData(i))
	}
	knot.device.Data = fakeData
	knot.publisherMock.On("PublishDeviceData").Return(errors.New("Communication error"))

	states := []string{entities.KnotPublishing}
	<-knot.pipeDevices
	for _, state := range states {
		knot.device.State = state
		deviceChan <- knot.device
		time.Sleep(1 * time.Second)
		var logMessages []string
		for _, message := range knot.hook.Entries {
			logMessages = append(logMessages, message.Message)
		}
		assert.Contains(knot.T(), logMessages, "Communication error")
	}
}

func TestRequestKNotSuite(t *testing.T) {
	suite.Run(t, new(requestKNotSuite))
}

type knotStateDeviceEmptyTokenSuite struct {
	suite.Suite
	hook          *test.Hook
	fakeDevice    entities.Device
	logger        *logrus.Entry
	publisherMock *mocks.PublisherMock
	deviceChan    chan entities.Device
	pipeDevices   chan map[string]entities.Device
	amqpMock      *network.AmqpMock
	protocol      Protocol
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) SetupTest() {
	os.Setenv("DEVICE_CONFIG_FILEPATH", "/tmp/device_config.yaml")
	deviceConfiguration, knotConfiguration := loadConfiguration()
	for key, value := range deviceConfiguration {
		value.Token = ""
		deviceConfiguration[key] = value
	}
	knotSuite.pipeDevices, knotSuite.deviceChan = createChannels()
	knotSuite.logger, knotSuite.hook = createNullLogger()
	msgChan := make(chan network.InMsg)
	knotSuite.publisherMock = new(mocks.PublisherMock)
	subscriberMock := new(mocks.SubscriberMock)
	knotSuite.amqpMock = new(network.AmqpMock)
	knotSuite.publisherMock.On("PublishDeviceRegister").Return(nil)
	subscriberMock.On("SubscribeToKNoTMessages", msgChan).Return(nil)
	fileManagementMock := new(fileManagementMock)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	var err error
	knotSuite.protocol, err = newProtocol(knotSuite.pipeDevices, knotConfiguration, knotSuite.deviceChan, msgChan, knotSuite.logger, deviceConfiguration, knotSuite.publisherMock, subscriberMock, knotSuite.amqpMock, fileManagementMock)

	assert.NoError(knotSuite.T(), err)
	knotSuite.fakeDevice = deviceConfiguration[deviceID]
	<-knotSuite.pipeDevices
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) TestKnotStateMachineHandlerSpecificStates() {
	for _, test := range testCasesRequestsKnotStates {
		knotSuite.publisherMock.On(test.functionName).Return(nil)
		knotSuite.fakeDevice.State = test.oldState
		knotSuite.deviceChan <- knotSuite.fakeDevice
		time.Sleep(1 * time.Second)
		var logMessages []string
		for _, message := range knotSuite.hook.Entries {
			logMessages = append(logMessages, message.Message)
		}
		assert.Contains(knotSuite.T(), logMessages, test.expectedMessage)
	}
}

type invalidDeviceID struct {
	id                   string
	expectedErrorMessage string
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) TestKnotStateMachineHandlerWhenNonExistentDeviceReturnError() {
	var invalidDevicesID []invalidDeviceID
	invalidDevicesID = append(invalidDevicesID, invalidDeviceID{id: "", expectedErrorMessage: "Generated device id"})
	invalidDevicesID = append(invalidDevicesID, invalidDeviceID{id: "invalid", expectedErrorMessage: "device id received does not match the stored"})

	for _, invalidDeviceID := range invalidDevicesID {
		knotSuite.fakeDevice.ID = invalidDeviceID.id
		knotSuite.deviceChan <- knotSuite.fakeDevice
		time.Sleep(1 * time.Second)
		var logMessages []string
		for _, message := range knotSuite.hook.Entries {
			logMessages = append(logMessages, message.Message)
		}
		assert.Contains(knotSuite.T(), logMessages, invalidDeviceID.expectedErrorMessage)
	}
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) TestKnotStateMachineHandlerForcedDeleteStateReturnDeleteMessage() {
	knotSuite.fakeDevice.State = entities.KnotForceDelete

	knotSuite.deviceChan <- knotSuite.fakeDevice
	time.Sleep(1 * time.Second)
	var logMessages []string
	for _, message := range knotSuite.hook.Entries {
		logMessages = append(logMessages, message.Message)
	}
	assert.Contains(knotSuite.T(), logMessages, "delete a device")
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) TestKnotStateMachineHandlerReadyState() {
	knotSuite.fakeDevice.State = entities.KnotReady

	knotSuite.deviceChan <- knotSuite.fakeDevice
	time.Sleep(1 * time.Second)
	devices := <-knotSuite.pipeDevices
	assert.Equal(knotSuite.T(), devices[deviceID].State, entities.KnotPublishing)
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) TestKnotStateMachineHandlerErrorStateReturnErrorMessage() {
	knotError := "thing's config not provided"
	knotSuite.fakeDevice.State = entities.KnotError

	var expectedErrorMessages []string = []string{knotError, "ERROR WITHOUT HANDLER" + "error"}
	var errorMessages []string = []string{knotError, "error"}
	for index, errorMessage := range errorMessages {
		knotSuite.fakeDevice.Error = errorMessage
		knotSuite.deviceChan <- knotSuite.fakeDevice
		time.Sleep(1 * time.Second)
		var logMessages []string
		for _, message := range knotSuite.hook.Entries {
			logMessages = append(logMessages, message.Message)
		}
		assert.Contains(knotSuite.T(), logMessages, expectedErrorMessages[index])
	}
}

func (knotSuite *knotStateDeviceEmptyTokenSuite) TestClose() {
	knotSuite.amqpMock.On("Stop").Return(nil)
	err := knotSuite.protocol.Close()
	assert.Nil(knotSuite.T(), err)
}

func TestKnotStateDeviceEmptyTokenSuite(t *testing.T) {
	suite.Run(t, new(knotStateDeviceEmptyTokenSuite))
}

func TestNewProtocolWhenSubscribeToKNoTMessagesErrorReturnErrorMessage(t *testing.T) {
	os.Setenv("DEVICE_CONFIG_FILEPATH", "/tmp/device_config.yaml")
	deviceConfiguration, knotConfiguration := loadConfiguration()
	for key, value := range deviceConfiguration {
		value.Token = ""
		deviceConfiguration[key] = value
	}
	pipeDevices, deviceChan := createChannels()
	logger, _ := createNullLogger()
	msgChan := make(chan network.InMsg)
	publisherMock := new(mocks.PublisherMock)
	subscriberMock := new(mocks.SubscriberMock)
	amqpMock := new(network.AmqpMock)
	errorToSubscribeMessage := "errorToSubscriber"
	subscriberMock.On("SubscribeToKNoTMessages", msgChan).Return(errors.New(errorToSubscribeMessage))
	fileManagementMock := new(fileManagementMock)
	fileManagementMock.On("writeDevicesConfigFile").Return(nil)
	_, err := newProtocol(pipeDevices, knotConfiguration, deviceChan, msgChan, logger, deviceConfiguration, publisherMock, subscriberMock, amqpMock, fileManagementMock)
	assert.ErrorContains(t, err, errorToSubscribeMessage)
}

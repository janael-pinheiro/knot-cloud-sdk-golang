package knot

import (
	"fmt"
	"os"
	"testing"

	bloomFilter "github.com/bits-and-blooms/bloom/v3"
	"github.com/janael-pinheiro/knot-cloud-sdk-golang/pkg/entities"
	"github.com/janael-pinheiro/knot-cloud-sdk-golang/pkg/gateways/knot/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const sensorID = 1
const testTimestamp = "2023-08-25"

type knotIntegrationSuite struct {
	suite.Suite
	integration         *Integration
	fakeDevice          entities.Device
	deviceConfiguration map[string]entities.Device
}

func (knot *knotIntegrationSuite) SetupTest() {
	knot.deviceConfiguration, _ = loadConfiguration()
	knot.fakeDevice = knot.deviceConfiguration[deviceID]
	knot.integration = new(Integration)
	knot.integration.filters = map[string]*bloomFilter.BloomFilter{}
	knot.integration.filters[knot.fakeDevice.ID] = bloomFilter.NewWithEstimates(1000000, 0.01)
	knot.integration.maximumPercentageFilterUsage = 0.75
}

func (knot *knotIntegrationSuite) TestHandleDevice() {
	go knot.integration.HandleDevice(knot.fakeDevice)
	device := <-deviceChan
	assert.Equal(knot.T(), knot.fakeDevice.Name, device.Name)
	assert.Equal(knot.T(), knot.fakeDevice.Token, device.Token)
}

func (knot *knotIntegrationSuite) TestGetDevice() {
	actualDevice := knot.integration.GetDevice(knot.deviceConfiguration)
	assert.Equal(knot.T(), knot.fakeDevice.Name, actualDevice.Name)
	assert.Equal(knot.T(), knot.fakeDevice.Token, actualDevice.Token)
}

func (knot *knotIntegrationSuite) TestTransmit() {
	knot.fakeDevice.State = entities.KnotPublishing
	var fakeData []entities.Data
	numberSensors := 2
	for i := 1; i <= numberSensors; i++ {
		fakeData = append(fakeData, createData(i))
	}
	knot.fakeDevice.Data = fakeData
	knot.integration.isMeasurementDuplicatedFunction = knot.integration.isMeasurementDuplicated
	go knot.integration.Transmit(knot.fakeDevice)
	device := <-deviceChan
	assert.Equal(knot.T(), knot.fakeDevice.Name, device.Name)
	assert.Equal(knot.T(), numberSensors, len(device.Data))
	assert.Empty(knot.T(), deviceChan)
}

func (knot *knotIntegrationSuite) TestTransmitWhenDuplicateDataOmitDuplication() {
	knot.fakeDevice.State = entities.KnotPublishing
	var fakeData []entities.Data
	numberSensors := 2
	for i := 1; i <= numberSensors; i++ {
		fakeData = append(fakeData, createData(sensorID))
	}
	knot.fakeDevice.Data = fakeData
	knot.integration.isMeasurementDuplicatedFunction = knot.integration.isMeasurementDuplicated
	go knot.integration.Transmit(knot.fakeDevice)
	device := <-deviceChan
	assert.Equal(knot.T(), knot.fakeDevice.Name, device.Name)
	assert.Equal(knot.T(), numberSensors-1, len(device.Data))
	assert.Empty(knot.T(), deviceChan)
}

func (knot *knotIntegrationSuite) TestSentDataToKNoT() {
	knot.fakeDevice.State = entities.KnotPublishing
	var fakeData []entities.Data
	numberSensors := 2
	for i := 1; i <= numberSensors; i++ {
		fakeData = append(fakeData, createData(i))
	}
	knot.integration.isMeasurementDuplicatedFunction = knot.integration.isMeasurementDuplicated
	go knot.integration.SentDataToKNoT(fakeData, knot.fakeDevice)
	device := <-deviceChan
	assert.Equal(knot.T(), knot.fakeDevice.Name, device.Name)
	assert.Equal(knot.T(), numberSensors, len(device.Data))
	assert.Empty(knot.T(), deviceChan)
}

func (knot *knotIntegrationSuite) TestRegister() {
	knot.fakeDevice.State = entities.KnotPublishing
	knot.deviceConfiguration[knot.fakeDevice.ID] = knot.fakeDevice
	numberSensors := 1
	knot.integration.pipeDevices = make(chan map[string]entities.Device)
	go func(pipeDevices chan map[string]entities.Device, deviceConfiguration map[string]entities.Device) {
		<-deviceChan
		pipeDevices <- deviceConfiguration
	}(knot.integration.pipeDevices, knot.deviceConfiguration)
	device := knot.integration.Register(knot.fakeDevice)
	assert.Equal(knot.T(), knot.fakeDevice.Name, device.Name)
	assert.Equal(knot.T(), numberSensors, len(device.Data))
	assert.Empty(knot.T(), deviceChan)
}

func (knot *knotIntegrationSuite) TestIsMeasurementDuplicatedWhenExistingMeasurementThenTrue() {
	knot.integration.filters[knot.fakeDevice.ID] = knot.integration.filters[knot.fakeDevice.ID].Add([]byte(fmt.Sprintf("%s_%d", testTimestamp, sensorID)))
	isMeasurementDuplicated := knot.integration.isMeasurementDuplicated(testTimestamp, sensorID, knot.fakeDevice.ID)
	assert.True(knot.T(), isMeasurementDuplicated)
}

func (knot *knotIntegrationSuite) TestIsMeasurementDuplicatedWhenDuplicateMeasurementThenTrue() {
	newTimestamp := "2023-08-26"
	knot.integration.filters[knot.fakeDevice.ID] = knot.integration.filters[knot.fakeDevice.ID].Add([]byte(fmt.Sprintf("%s_%d", testTimestamp, sensorID)))
	isMeasurementDuplicated := knot.integration.isMeasurementDuplicated(newTimestamp, sensorID, knot.fakeDevice.ID)
	assert.False(knot.T(), isMeasurementDuplicated)
	knot.integration.filters[knot.fakeDevice.ID] = knot.integration.updateDuplicationFilter(newTimestamp, sensorID, knot.fakeDevice.ID)
	isMeasurementDuplicated = knot.integration.isMeasurementDuplicated(testTimestamp, sensorID, knot.fakeDevice.ID)
	assert.True(knot.T(), isMeasurementDuplicated)
}

func (knot *knotIntegrationSuite) TestIsMeasurementDuplicatedWhenNewMeasurementThenFalse() {
	isMeasurementDuplicated := knot.integration.isMeasurementDuplicated("2023-08-26", sensorID, knot.fakeDevice.ID)
	assert.False(knot.T(), isMeasurementDuplicated)
}

func (knot *knotIntegrationSuite) TestUpdateDuplicationFilter() {
	newTimestamp := "2023-08-29"
	knot.integration.filters[knot.fakeDevice.ID] = knot.integration.updateDuplicationFilter(newTimestamp, sensorID, knot.fakeDevice.ID)
	assert.True(knot.T(), knot.integration.filters[knot.fakeDevice.ID].Test([]byte(fmt.Sprintf("%s_%d", newTimestamp, sensorID))))
}

func (knot *knotIntegrationSuite) TestResetDuplicationFilter() {
	filterCapacity := knot.integration.filters[knot.fakeDevice.ID].Cap()
	maximumAllowedFilterSize := float32(filterCapacity) * 0.80
	for i := 0; i < int(maximumAllowedFilterSize); i++ {
		knot.integration.filters[knot.fakeDevice.ID].Add([]byte(fmt.Sprintf("%d", i)))
	}
	assert.NotEqual(knot.T(), 0, int(knot.integration.filters[knot.fakeDevice.ID].ApproximatedSize()))
	knot.integration.resetDuplicationFilter(knot.fakeDevice.ID)
	assert.Equal(knot.T(), 0, int(knot.integration.filters[knot.fakeDevice.ID].ApproximatedSize()))
}

func TestKnotIntegrationSuite(t *testing.T) {
	suite.Run(t, new(knotIntegrationSuite))
}

func TestClose(t *testing.T) {
	amqpMock := new(network.AmqpMock)
	amqpMock.On("Stop", msgChan).Return(nil)
	net := new(networkWrapper)
	net.amqp = make([]network.Messaging, 1)
	net.amqp[0] = amqpMock
	protocol := protocol{
		network: net,
	}
	knotIntegration := new(Integration)
	knotIntegration.protocol = &protocol
	err := knotIntegration.Close()
	assert.NoError(t, err)
}

func TestGetValueFromEnvironmentVariableWhenVariableExistsThenReturnValue(t *testing.T) {
	variableName := "TEST_VARIABLE"
	expectedVariableValue := "0"
	defaultValue := "1"
	os.Setenv(variableName, expectedVariableValue)
	actualVariableValue := getValueFromEnvironmentVariable(variableName, defaultValue)
	assert.Equal(t, expectedVariableValue, actualVariableValue)
}

func TestGetValueFromEnvironmentVariableWhenVariableNotExistsThenReturnDefaultValue(t *testing.T) {
	variableName := "TEST_VARIABLE_2"
	defaultValue := "1"
	actualVariableValue := getValueFromEnvironmentVariable(variableName, defaultValue)
	assert.Equal(t, defaultValue, actualVariableValue)
}

package knot

import (
	"testing"

	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network"
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
	knot.integration.sensorIDTimestampMapping = make(map[int]string)
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
	knot.integration.sensorIDTimestampMapping = make(map[int]string)
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
	knot.integration.sensorIDTimestampMapping = make(map[int]string)
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
	knot.integration.sensorIDTimestampMapping = make(map[int]string)
	go func(pipeDevices chan map[string]entities.Device, deviceConfiguration map[string]entities.Device) {
		<-deviceChan
		pipeDevices <- deviceConfiguration
	}(knot.integration.pipeDevices, knot.deviceConfiguration)
	device := knot.integration.Register(knot.fakeDevice)
	assert.Equal(knot.T(), knot.fakeDevice.Name, device.Name)
	assert.Equal(knot.T(), numberSensors, len(device.Data))
	assert.Empty(knot.T(), deviceChan)
}

func TestKnotIntegrationSuite(t *testing.T) {
	suite.Run(t, new(knotIntegrationSuite))
}

func TestClose(t *testing.T) {
	amqpMock := new(network.AmqpMock)
	amqpMock.On("Stop", msgChan).Return(nil)
	net := new(networkWrapper)
	net.amqp = amqpMock
	protocol := protocol{
		network: net,
	}
	knotIntegration := new(Integration)
	knotIntegration.protocol = &protocol
	err := knotIntegration.Close()
	assert.NoError(t, err)
}

func TestIsMeasurementNewWhenExistingMeasurementThenFalse(t *testing.T) {
	timestampMapping := map[int]string{sensorID: testTimestamp}
	isMeasurementNew := isMeasurementNew(timestampMapping, "2023-08-25", sensorID)
	assert.False(t, isMeasurementNew)
}

func TestIsMeasurementNewWhenNewMeasurementThenTrue(t *testing.T) {
	timestampMapping := map[int]string{sensorID: testTimestamp}
	isMeasurementNew := isMeasurementNew(timestampMapping, "2023-08-26", sensorID)
	assert.True(t, isMeasurementNew)
}

func TestUpdateTagNameTimestampMapping(t *testing.T) {
	timestampMapping := map[int]string{sensorID: testTimestamp}
	newTimestamp := "2023-08-29"
	updatedTagNameTimestampMapping := updateSensorIDTimestampMapping(timestampMapping, newTimestamp, sensorID)
	assert.Equal(t, updatedTagNameTimestampMapping[sensorID], newTimestamp)
}

# KNoT Cloud SDK Go 
This is a KNoT Cloud Go client.

## Usage

To use this client in your application, you need to import it:
```go
import (
    "github.com/janael-pinheiro/knot-cloud-sdk-golang/pkg/entities"
    "github.com/janael-pinheiro/knot-cloud-sdk-golang/pkg/gateways/knot"
)
```

Create a channel for transmitting the mapping with the devices configuration and instantiate the integration with KNoT:

```go
knotConfiguration := entities.IntegrationKNoTConfig{
    UserToken: "<knot_token>",
	URL: "amqp://<username>:<password>@IP:5672",           
}

deviceConfiguration := make(map[string]entities.Device)
deviceConfiguration["0d7cd9d221385e1f"] = new(entities.Device)

log := logging.NewLogrus("info", os.Stdout)
logger := log.Get("Main")

pipeDevices := make(chan map[string]entities.Device)
knotIntegration, err := knot.NewKNoTIntegration(pipeDevices, knotConfiguration, logger, deviceConfiguration)
```

Device configuration in **deviceConfiguration** must be valid. Wait for device mapping validated by KNoT SDK. The GetDevice() method returns the device present in the mapping. In turn, the Register method performs the necessary procedures to ensure that the device is ready to send data to the KNoT Cloud.

```go
devices := <-pipeDevices
device := knotIntegration.GetDevice(devices)
device = knotIntegration.Register(device)
```

The **SentDataToKNoT** is used to send data to the KNoT Cloud. This method receives a slice of entities.Data and an entities.Device as arguments.

```go
var sensors []entities.Data
sensor := entities.Data{SensorID: 1, Value: 73, TimeStamp: "2023-08-26 20:54:18"}
sensors = append(sensors, sensor)

knotIntegration.SentDataToKNoT(sensors, device)
//Reset the sensors array to avoid data duplication.
device.Data = nil
sensors = nil
```

## Configuration
### Duplication filter
For reasons of efficiency and resource usage, we implemented a bloom filter to mitigate the occurrence of duplicate measurements. Bloom filter is a probabilistic data structure, which, in this case, allows us to implement a mechanism to avoid data duplication with a low memory consumption (multiple records of the same combination of device, sensor and timestamp). This mechanism effectively prevents duplication. However, there is a cost, due to the nature of the bloom filter, there is the possibility that non-duplicated measurements are mistakenly identified as duplicates, which prevents this data from being sent to KNoT.

You can set the probability of false positives occurring in the duplication filter by setting the value of the DUPLICATION_PROBABILITY environment variable. The default value of this probability is 0.01. You can also set the filter capacity through the FILTER_CAPACITY environment variable. The default capacity is 1,000,000. The filter, with a capacity of 1,000,000 items and a false positive probability of 1%, consumes approximately less than 2 MB of memory. You can estimate the memory consumption in this link: [Bloom Filter Calculator](https://hur.st/bloomfilter/).

Due to the boom filter, the more measurements in the filter, the more likely false positives will occur. Therefore, the duplication filter is reset from time to time when usage exceeds a certain threshold. You can set this threshold in percentage form with the RESET_FILTER_USAGE_PERCENTAGE variable. The default threshold is 75%. That way, by default, when the filter reaches 75% usage it is reset. This means that older measurements can be sent back to the KNoT as the filter loses knowledge that they have already been sent. 

Each device has its own filter. You can enable or disable the duplication filter through the DUPLICATION_FILTER variable. By default, this filter is disabled. 
- Enable: DUPLICATION_FILTER=1
- Disable: DUPLICATION_FILTER=0

Default setting:
- DUPLICATION_FILTER=0
- FILTER_CAPACITY=1000000
- DUPLICATION_PROBABILITY=0.01
- RESET_FILTER_USAGE_PERCENTAGE=0.75

## Device configuration file
The DEVICE_CONFIG_FILEPATH environment variable must be set to the path of the file containing the devices configuration.

## Environment variables
- DEVICE_CONFIG_FILEPATH
- DUPLICATION_FILTER
- FILTER_CAPACITY
- DUPLICATION_PROBABILITY
- RESET_FILTER_USAGE_PERCENTAGE


## Tests
```sh
$ go test -coverprofile cover.out ./...
$ go tool cover -html=cover.out -o cover.html
```

## Benchmarks
```sh
$ go test -bench=. -count 10 -run=^# ./...
```

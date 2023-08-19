package entities

const (
	KnotNew         string = "new"
	KnotAlreadyReg  string = "alreadyRegistered"
	KnotRegistered  string = "registered"
	KnotForceDelete string = "forceDelete"
	KnotReady       string = "readyToSendData"
	KnotPublishing  string = "SendData"
	KnotAuth        string = "authenticated"
	KnotError       string = "error"
	KnotWaitReg     string = "waitResponseRegister"
	KnotWaitAuth    string = "waitResponseAuth"
	KnotWaitConfig  string = "waitResponseConfig"
	KnotOff         string = "ignore"
)

type Device struct {
	ID     string   `yaml:"id"`
	Token  string   `yaml:"token"`
	Name   string   `yaml:"name"`
	Config []Config `yaml:"config"`
	State  string   `yaml:"state"`
	Data   []Data   `yaml:"data"`
	Error  string
}

type Sensor struct {
	ID    int
	Name  string
	Value float32
}

type Config struct {
	SensorID int    `yaml:"sensorId"`
	Schema   Schema `yaml:"schema"`
	Event    Event  `yaml:"event"`
}

type Schema struct {
	ValueType int    `yaml:"valueType"`
	Unit      int    `yaml:"unit"`
	TypeID    int    `yaml:"typeId"`
	Name      string `yaml:"name"`
}

type Event struct {
	Change         bool        `yaml:"change"`
	TimeSec        int         `yaml:"timeSec"`
	LowerThreshold interface{} `yaml:"lowerThreshold"`
	UpperThreshold interface{} `yaml:"upperThreshold"`
}

type Data struct {
	SensorID  int         `json:"sensorId"`
	Value     interface{} `json:"value"`
	TimeStamp interface{} `json:"timestamp"`
}

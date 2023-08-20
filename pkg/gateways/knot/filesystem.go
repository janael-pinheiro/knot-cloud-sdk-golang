package knot

import "os"

type filesystemManagement interface {
	writeDevicesConfigFile(filepath string, data []byte) error
}

type fileManagement struct{}

func (fs *fileManagement) writeDevicesConfigFile(filepath string, data []byte) error {
	return os.WriteFile(os.Getenv(filepath), data, 0600)
}
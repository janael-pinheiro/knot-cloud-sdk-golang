package knot

import "github.com/stretchr/testify/mock"

type fileManagementMock struct {
	mock.Mock
}

func (fm *fileManagementMock) writeDevicesConfigFile(filepath string, data []byte) error {
	args := fm.Called()
	return args.Error(0)
}

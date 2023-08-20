package mocks

import (
	"github.com/janael-pinheiro/knot_go_sdk/pkg/gateways/knot/network"
	"github.com/stretchr/testify/mock"
)

type SubscriberMock struct {
	mock.Mock
}

func (s *SubscriberMock) SubscribeToKNoTMessages(msgChan chan network.InMsg) error {
	args := s.Called(msgChan)
	return args.Error(0)
}

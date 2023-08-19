package network

import (
	"github.com/janael-pinheiro/knot_go_sdk/pkg/entities"
)

type DeviceGenericMessage struct {
	ID     string            `json:"id"`
	Name   string            `json:"name,omitempty"`
	Token  string            `json:"token,omitempty"`
	Error  string            `json:"error,omitempty"`
	Config []entities.Config `json:"config,omitempty"`
	Data   []entities.Data   `json:"data,omitempty"`
}

type DeviceRegisterRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeviceRegisteredResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
	Error string `json:"error"`
}

type DeviceUnregisterRequest struct {
	ID string `json:"id"`
}

type DeviceUnregisteredResponse struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type ConfigUpdateRequest struct {
	ID     string            `json:"id"`
	Config []entities.Config `json:"config,omitempty"`
}

type ConfigUpdatedResponse struct {
	ID      string            `json:"id"`
	Config  []entities.Config `json:"config,omitempty"`
	Changed bool              `json:"changed"`
	Error   string            `json:"error"`
}

type DeviceAuthRequest struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type DeviceAuthResponse struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

type DataSent struct {
	ID   string          `json:"id"`
	Data []entities.Data `json:"data"`
}

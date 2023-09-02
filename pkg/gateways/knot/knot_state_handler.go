package knot

import (
	"github.com/janael-pinheiro/knot-cloud-sdk-golang/pkg/entities"
	"github.com/sirupsen/logrus"
)

type stateHandler interface {
	execute(entities.Device)
	setNext(stateHandler)
}

type baseState struct {
	next        stateHandler
	p           *protocol
	deviceChan  chan entities.Device
	log         *logrus.Entry
	pipeDevices chan map[string]entities.Device
}

func (bs *baseState) execute(device entities.Device) {}

func (bs *baseState) setNext(next stateHandler) {
	bs.next = next
}

type newStateHandler struct {
	baseState
}

func (ns *newStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotNew {
		ns.p.requestsKnot(ns.deviceChan, device, device.State, entities.KnotWaitReg, registerRequestMessage, ns.log)
	} else {
		ns.next.execute(device)
	}
}

type registeredStateHandler struct {
	baseState
}

func (ns *registeredStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotRegistered {
		ns.p.requestsKnot(ns.deviceChan, device, device.State, entities.KnotWaitAuth, "send a auth request", ns.log)
	} else {
		ns.next.execute(device)
	}
}

type authStateHandler struct {
	baseState
}

func (ns *authStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotAuth {
		ns.p.requestsKnot(ns.deviceChan, device, device.State, entities.KnotWaitConfig, "send a updateconfig request", ns.log)
	} else {
		ns.next.execute(device)
	}
}

type readyStateHandler struct {
	baseState
}

func (ns *readyStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotReady {
		device.State = entities.KnotPublishing
		knotMutex.Lock()
		err := ns.p.updateDevice(device)
		knotMutex.Unlock()
		if err != nil {
			ns.log.Errorln(err)
		} else {
			go updateDeviceMap(ns.pipeDevices, ns.p.devices)
		}
	} else {
		ns.next.execute(device)
	}
}

type publishingStateHandler struct {
	baseState
}

func (ns *publishingStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotPublishing {
		err := ns.p.checkData(device)
		if err == nil {
			err := ns.p.network.publisher.PublishDeviceData(ns.p.userToken, &device, device.Data)
			if err != nil {
				ns.log.Errorln(err)
			} else {
				ns.log.Println("Published data")
				device.Data = nil
				knotMutex.Lock()
				err = ns.p.updateDevice(device)
				knotMutex.Unlock()
				verifyErrors(err, ns.log)
			}
		} else {
			ns.log.Println("invalid data, has no data to send")
		}
	} else {
		ns.next.execute(device)
	}
}

type alreadyRegStateHandler struct {
	baseState
}

func (ns *alreadyRegStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotAlreadyReg {
		var err error
		if device.Token == "" {
			knotMutex.Lock()
			device.ID, err = ns.p.generateID(ns.pipeDevices, device)
			knotMutex.Unlock()
			if err != nil {
				ns.log.Error(err)
			} else {
				ns.p.requestsKnot(deviceChan, device, entities.KnotNew, entities.KnotWaitReg, registerRequestMessage, ns.log)
			}
		} else {

			ns.p.requestsKnot(deviceChan, device, entities.KnotRegistered, entities.KnotWaitAuth, "send a Auth request", ns.log)

		}
	} else {
		ns.next.execute(device)
	}
}

type forceDeleteStateHandler struct {
	baseState
}

func (ns *forceDeleteStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotForceDelete {
		var err error
		ns.log.Println("delete a device")
		knotMutex.Lock()
		device.ID, err = ns.p.generateID(ns.pipeDevices, device)
		knotMutex.Unlock()
		if err != nil {
			ns.log.Error(err)
		} else {
			ns.p.requestsKnot(deviceChan, device, entities.KnotNew, entities.KnotWaitReg, registerRequestMessage, ns.log)
		}
	} else {
		ns.next.execute(device)
	}
}

type errorStateHandler struct {
	baseState
}

func (ns *errorStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotError {
		ns.log.Println("ERROR: ")
		switch device.Error {
		case "thing's config not provided":
			ns.log.Println("thing's config not provided")

		default:
			ns.log.Println("ERROR WITHOUT HANDLER" + device.Error)

		}
		device.State = entities.KnotNew
		device.Error = ""
		knotMutex.Lock()
		err := ns.p.updateDevice(device)
		knotMutex.Unlock()
		verifyErrors(err, ns.log)
	} else {
		ns.next.execute(device)
	}
}

type knotOffStateHandler struct {
	baseState
}

func (ns *knotOffStateHandler) execute(device entities.Device) {
	if device.State == entities.KnotOff {
		ns.log.Println("KNoT device off.")
	} else {
		ns.next.execute(device)
	}
}

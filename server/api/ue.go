package api

import "fmt"

type UeApi struct {
	// Pointer to backend engine
}

func CreateUeApi() *UeApi {
	return &UeApi{}
}

func (uApi *UeApi) Register(isEmergency bool) bool {
	emergency := ""
	if isEmergency {
		emergency = "emergency "
	}
	fmt.Printf("UE registering with %sservices\n", emergency)
	// Implement registration logic
	return true
}

func (uApi *UeApi) Deregister(deregisterType uint8) bool {
	fmt.Printf("UE deregistering with type: %d\n", deregisterType)
	// Implement deregistration logic
	return true
}

func (uApi *UeApi) CreateSession(slice string, dnName string, sessionType uint8) bool {
	fmt.Printf("Creating session with slice: %s and DN name: %s of type: %d\n",
		slice, dnName, sessionType)
	// Implement session creation logic
	return true
}

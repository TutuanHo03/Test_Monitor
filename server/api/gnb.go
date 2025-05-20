package api

import "fmt"

type GnbApi struct {
	// Pointer to backend engine
}

func CreateGnbApi() *GnbApi {
	return &GnbApi{}
}

func (gApi *GnbApi) ReleaseUe(ueId string) bool {
	fmt.Printf("Releasing UE with ID: %s from gNB\n", ueId)
	// Implement UE release logic
	return true
}

func (gApi *GnbApi) ReleaseSession(ueId string, sessionId uint8) bool {
	fmt.Printf("Releasing session %d for UE: %s from gNB\n", sessionId, ueId)
	// Implement session release logic
	return true
}

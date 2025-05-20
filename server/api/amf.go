package api

import "fmt"

type AmfApi struct {
	// Pointer to backend AMF engine
	ueContexts []string
}

func CreateAmfApi() *AmfApi {
	return &AmfApi{
		ueContexts: []string{"imsi-123456789012345", "imsi-234567890123456", "imsi-345678901234567"},
	}
}

func (aApi *AmfApi) ListUeContexts() []string {
	fmt.Println("Listing all UE contexts in AMF")
	return aApi.ueContexts
}

func (aApi *AmfApi) DeregisterUe(imsi string, cause uint8) bool {
	fmt.Printf("AMF deregistering UE with IMSI: %s and cause: %d\n", imsi, cause)

	// Simulate deregistration by removing from UE contexts
	for i, ue := range aApi.ueContexts {
		if ue == imsi {
			// Remove the UE from the list
			aApi.ueContexts = append(aApi.ueContexts[:i], aApi.ueContexts[i+1:]...)
			return true
		}
	}

	// UE not found in contexts
	return false
}

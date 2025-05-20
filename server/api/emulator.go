package api

import "fmt"

type EmulatorApi struct {
	ues  []string
	gnbs []string
}

func CreateEmulatorApi() *EmulatorApi {
	return &EmulatorApi{
		ues:  []string{"ue1", "ue2", "ue3"},
		gnbs: []string{"gnb1", "gnb2"},
	}
}

func (eApi *EmulatorApi) ListUes() []string {
	return eApi.ues
}

func (eApi *EmulatorApi) ListGnbs() []string {
	return eApi.gnbs
}

func (eApi *EmulatorApi) AddUe(supi string, triggerRegister bool) bool {
	register := ""
	if triggerRegister {
		register = " with auto-registration"
	}
	fmt.Printf("Adding UE with SUPI: %s%s\n", supi, register)
	// Add to list if not exists
	for _, ue := range eApi.ues {
		if ue == supi {
			return true
		}
	}
	eApi.ues = append(eApi.ues, supi)
	return true
}

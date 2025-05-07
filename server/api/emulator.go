package api

type EmulatorApi struct {
	// Pointer to backend engine to help implementing APIs
}

func CreateEmulatorApi() *EmulatorApi {
	return &EmulatorApi{}
}

func (eApi *EmulatorApi) ListUes() []string {
	// Implemented by Mssim
	return []string{"ue1", "ue2"}
}

func (eApi *EmulatorApi) ListGnbs() []string {
	// Implemented by Mssim
	return []string{"gnb1", "gnb2"}
}

func (eApi *EmulatorApi) AddUe(supi string, triggerRegister bool) bool {
	// Implemented by Mssim
	return true
}

package api

type EmulatorApi struct {
	//a pointer to backend engine to help implementing APIs
}

func CreateEmulatorApi( /*add pointer to backend engine*/ ) *EmulatorApi {
	return &EmulatorApi{}
}

func (eApi *EmulatorApi) ListUes() []string {
	return []string{"ue1", "ue2"}
}

func (eApi *EmulatorApi) ListGnbs() []string {
	return []string{"gnb1", "gnb2"}
}

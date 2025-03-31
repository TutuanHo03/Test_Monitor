package api

type GnbApi struct {
	// Pointer to backend engine
}

func CreateGnbApi() *GnbApi {
	return &GnbApi{}
}

func (gApi *GnbApi) ReleaseUe(ueId string) bool {
	// Implement UE release logic
	return true
}

func (gApi *GnbApi) ReleaseSession(ueId string, sessionId uint8) bool {
	// Implement session release logic
	return true
}

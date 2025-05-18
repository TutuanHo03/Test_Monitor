package api

type AmfApi struct {
	// Pointer to backend AMF engine
}

func CreateAmfApi() *AmfApi {
	return &AmfApi{}
}

func (aApi *AmfApi) ListUeContexts() []string { // quan trọng
	// Would interface with the actual AMF service
	return []string{"imsi-123456789012345", "imsi-234567890123456"}
}

// view UE context
// AMF có thể trigger 1 event về PDU session
func (aApi *AmfApi) RegisterUe(imsi string) bool { //bỏ qua
	// Interface with AMF registration service
	return true
}

func (aApi *AmfApi) DeregisterUe(imsi string, cause uint8) bool { // quan trọng trigger AMF deregistration
	// Interface with AMF to deregister UE
	return true
}

func (aApi *AmfApi) GetServiceStatus() map[string]string {
	// Return service status information
	return map[string]string{
		"status":      "running",
		"uptime":      "10h30m",
		"connections": "5",
	}
}

func (aApi *AmfApi) GetConfiguration() map[string]interface{} {
	// Return AMF configuration information
	return map[string]interface{}{
		"amfSet":     "1-1",
		"amfPointer": 1,
		"plmnId": map[string]string{
			"mcc": "208",
			"mnc": "93",
		},
	}
}

// N1/N2 Interface Management
func (aApi *AmfApi) SendN1N2Message(ueId string, messageType string, content string) bool {
	// Send N1/N2 message to a UE
	return true
}

func (aApi *AmfApi) ListN1N2Subscriptions(ueId string) []string {
	// List N1/N2 message subscriptions for a UE
	return []string{"subscription-1", "subscription-2"}
}

// Handover Management
func (aApi *AmfApi) InitiateHandover(ueId string, targetGnb string) bool {
	// Initiate handover procedure
	return true
}

func (aApi *AmfApi) ListHandoverHistory(ueId string) []map[string]string {
	// List handover history for a UE
	return []map[string]string{
		{"time": "2023-09-01T10:00:00Z", "source": "gnb1", "target": "gnb2", "status": "success"},
		{"time": "2023-09-02T11:30:00Z", "source": "gnb2", "target": "gnb3", "status": "success"},
	}
}

// SBI Interface
func (aApi *AmfApi) GetNfSubscriptions() []string {
	// Get list of NF subscriptions
	return []string{"subscription-1", "subscription-2"}
}

func (aApi *AmfApi) GetSbiEndpoints() map[string]string {
	// Get SBI endpoints
	return map[string]string{
		"namf-comm": "http://localhost:8080/namf-comm/v1",
		"namf-evts": "http://localhost:8080/namf-evts/v1",
		"namf-loc":  "http://localhost:8080/namf-loc/v1",
		"namf-mt":   "http://localhost:8080/namf-mt/v1",
		"namf-oam":  "http://localhost:8080/namf-oam/v1",
	}
}

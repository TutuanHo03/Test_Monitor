package api

import "testing"

// EmulatorAPI test
func TestEmulatorApi_ListUes(t *testing.T) {
	api := CreateEmulatorApi()
	ues := api.ListUes()

	if len(ues) == 0 {
		t.Errorf("Expected at least one UE, got %d", len(ues))
	}

	expectedUes := map[string]bool{
		"ue1": true,
		"ue2": true,
		"ue3": true,
	}
	for _, ue := range ues {
		if !expectedUes[ue] {
			t.Errorf("Unexpected UE in list: %s", ue)
		}
	}

}

func TestEmulatorApi_ListGnbs(t *testing.T) {
	api := CreateEmulatorApi()
	gnbs := api.ListGnbs()

	if len(gnbs) == 0 {
		t.Errorf("Expected at least one gNB, got %d", len(gnbs))
	}

	expectedGnbs := map[string]bool{
		"gnb1": true,
		"gnb2": true,
	}
	for _, gnb := range gnbs {
		if !expectedGnbs[gnb] {
			t.Error("Expected gNBs to be returned, got empty list")
		}

	}
}

func TestEmulatorApi_AddUe(t *testing.T) {
	api := CreateEmulatorApi()

	testCases := []struct {
		supi            string
		triggerRegister bool
		expectedResult  bool
	}{
		{"imsi-208930000000003", true, true},
		{"imsi-208930000000004", false, true},
	}

	for _, tc := range testCases {
		result := api.AddUe(tc.supi, tc.triggerRegister)
		if result != tc.expectedResult {
			t.Errorf("AddUe(%s, %v) = %v; want %v",
				tc.supi, tc.triggerRegister, result, tc.expectedResult)
		}
	}
}

func TestUeApi_Register(t *testing.T) {
	api := CreateUeApi()
	//Test normal registration
	result := api.Register(false)
	if !result {
		t.Error("Expected registration (false) to succeed")
	}

	// Test emergency registration
	result = api.Register(true)
	if !result {
		t.Error("Expected Register(true) to succeed, but it failed")
	}

}

func TestUeApi_Deregister(t *testing.T) {
	api := CreateUeApi()

	// Test different deregistration types
	testCases := []struct {
		deregType      uint8
		expectedResult bool
	}{
		{0, true}, // Normal deregistration
		{1, true}, // Switch off
		{2, true}, // Combined EPS/IMSI detach
	}

	for _, tc := range testCases {
		result := api.Deregister(tc.deregType)
		if result != tc.expectedResult {
			t.Errorf("Deregister(%d) = %v; want %v",
				tc.deregType, result, tc.expectedResult)
		}
	}
}

func TestUeApi_CreateSession(t *testing.T) {
	api := CreateUeApi()

	// Test different session configurations
	testCases := []struct {
		slice          string
		dnName         string
		sessionType    uint8
		expectedResult bool
	}{
		{"slice1", "internet", 1, true},
		{"slice2", "ims", 2, true},
		{"slice3", "mms", 3, true},
	}

	for _, tc := range testCases {
		result := api.CreateSession(tc.slice, tc.dnName, tc.sessionType)
		if result != tc.expectedResult {
			t.Errorf("CreateSession(%s, %s, %d) = %v; want %v",
				tc.slice, tc.dnName, tc.sessionType, result, tc.expectedResult)
		}
	}
}

// GnbAPI Tests

func TestGnbApi_ReleaseUe(t *testing.T) {
	api := CreateGnbApi()

	// Test releasing different UEs
	testCases := []struct {
		ueId           string
		expectedResult bool
	}{
		{"ue1", true},
		{"ue2", true},
		{"nonexistent-ue", true}, // API currently returns true for all cases
	}

	for _, tc := range testCases {
		result := api.ReleaseUe(tc.ueId)
		if result != tc.expectedResult {
			t.Errorf("ReleaseUe(%s) = %v; want %v",
				tc.ueId, result, tc.expectedResult)
		}
	}
}

func TestGnbApi_ReleaseSession(t *testing.T) {
	api := CreateGnbApi()

	// Test releasing different sessions
	testCases := []struct {
		ueId           string
		sessionId      uint8
		expectedResult bool
	}{
		{"ue1", 1, true},
		{"ue2", 2, true},
		{"ue1", 3, true},
	}

	for _, tc := range testCases {
		result := api.ReleaseSession(tc.ueId, tc.sessionId)
		if result != tc.expectedResult {
			t.Errorf("ReleaseSession(%s, %d) = %v; want %v",
				tc.ueId, tc.sessionId, result, tc.expectedResult)
		}
	}
}

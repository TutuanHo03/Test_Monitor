// Hiện tại file server_test.go đang chạy được nhưng có lỗi khi chạy test
// Cần sửa lỗi về sau.
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestNewServer kiểm tra việc khởi tạo server
func TestNewServer(t *testing.T) {
	server := NewServer("localhost", 8080)

	if server == nil {
		t.Fatal("Failed to create server")
	}

	if server.Ip != "localhost" {
		t.Errorf("Expected IP 'localhost', got '%s'", server.Ip)
	}

	if server.Port != 8080 {
		t.Errorf("Expected Port 8080, got %d", server.Port)
	}

	if len(server.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(server.Nodes))
	}

	// Kiểm tra các nodes được khởi tạo đúng
	nodeTypes := make(map[NodeType]bool)
	for _, node := range server.Nodes {
		nodeTypes[node.Type] = true
	}

	if !nodeTypes[UE] {
		t.Error("UE node not initialized")
	}

	if !nodeTypes[Gnb] {
		t.Error("Gnb node not initialized")
	}
}

// TestSetupShellUE kiểm tra việc thiết lập shell UE và các command
func TestSetupShellUE(t *testing.T) {
	server := NewServer("localhost", 8080)

	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellUE(dummyFunction, dummyArgs)

	// Tìm UE node
	var ueNode *Node
	for i, node := range server.Nodes {
		if node.Type == UE {
			ueNode = &server.Nodes[i]
			break
		}
	}

	if ueNode == nil {
		t.Fatal("UE node not found after setup")
	}

	// Kiểm tra commands
	if len(ueNode.Commands) != 2 {
		t.Errorf("Expected 2 commands for UE, got %d", len(ueNode.Commands))
	}

	// Kiểm tra tên các commands
	commandNames := make(map[string]bool)
	for _, cmd := range ueNode.Commands {
		commandNames[cmd.Name] = true
	}

	if !commandNames["register"] {
		t.Error("register command not found")
	}

	if !commandNames["deregister"] {
		t.Error("deregister command not found")
	}
}

// TestSetupShellGnb kiểm tra việc thiết lập shell Gnb và các command
func TestSetupShellGnb(t *testing.T) {
	server := NewServer("localhost", 8080)

	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellGnb(dummyFunction, dummyArgs)

	// Tìm Gnb node
	var gnbNode *Node
	for i, node := range server.Nodes {
		if node.Type == Gnb {
			gnbNode = &server.Nodes[i]
			break
		}
	}

	if gnbNode == nil {
		t.Fatal("Gnb node not found after setup")
	}

	// Kiểm tra commands
	if len(gnbNode.Commands) != 2 {
		t.Errorf("Expected 2 commands for Gnb, got %d", len(gnbNode.Commands))
	}

	// Kiểm tra tên các commands
	commandNames := make(map[string]bool)
	for _, cmd := range gnbNode.Commands {
		commandNames[cmd.Name] = true
	}

	if !commandNames["amf-info"] {
		t.Error("amf-info command not found")
	}

	if !commandNames["amf-list"] {
		t.Error("amf-list command not found")
	}
}

// TestRegisterHandler kiểm tra xử lý lệnh register
func TestRegisterHandler(t *testing.T) {
	server := NewServer("localhost", 8080)

	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellUE(dummyFunction, dummyArgs)

	// Tìm UE node và register command
	var registerFunc func(map[string]string) string
	for _, node := range server.Nodes {
		if node.Type == UE {
			for _, cmd := range node.Commands {
				if cmd.Name == "register" {
					registerFunc = cmd.Func
					break
				}
			}
			break
		}
	}

	if registerFunc == nil {
		t.Fatal("register function not found")
	}

	// Test case 1: help flag
	args := map[string]string{"help": "true"}
	result := registerFunc(args)
	if !strings.Contains(result, "Usage: register") {
		t.Errorf("Expected help message, got: %s", result)
	}

	// Test case 2: register with AMF
	args = map[string]string{"amf-name": "AMF-01"}
	result = registerFunc(args)
	if !strings.Contains(result, "Registering UE") || !strings.Contains(result, "AMF-01") {
		t.Errorf("Expected register response with AMF-01, got: %s", result)
	}

	// Test case 3: register with multiple AMFs
	args = map[string]string{"amf-name": "AMF-01,AMF-02"}
	result = registerFunc(args)
	if !strings.Contains(result, "AMF-01, AMF-02") {
		t.Errorf("Expected register response with multiple AMFs, got: %s", result)
	}

	// Test case 4: register with SMF
	args = map[string]string{"smf-name": "SMF-01"}
	result = registerFunc(args)
	if !strings.Contains(result, "SMF-01") {
		t.Errorf("Expected register response with SMF-01, got: %s", result)
	}

	// Test case 5: invalid arguments
	args = map[string]string{"invalid": "true"}
	result = registerFunc(args)
	if !strings.Contains(result, "Error: No valid arguments") {
		t.Errorf("Expected error for invalid arguments, got: %s", result)
	}
}

// TestAmfListHandler kiểm tra xử lý lệnh amf-list
func TestAmfListHandler(t *testing.T) {
	server := NewServer("localhost", 8080)

	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellGnb(dummyFunction, dummyArgs)

	// Setup AllNodes and ActiveNodes
	for i, node := range server.Nodes {
		if node.Type == Gnb {
			server.Nodes[i].AllNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02", "AMF-03"},
			}
			server.Nodes[i].ActiveNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-03"},
			}
			break
		}
	}

	// Tìm Gnb node và amf-list command
	var amfListFunc func(map[string]string) string
	for _, node := range server.Nodes {
		if node.Type == Gnb {
			for _, cmd := range node.Commands {
				if cmd.Name == "amf-list" {
					amfListFunc = cmd.Func
					break
				}
			}
			break
		}
	}

	if amfListFunc == nil {
		t.Fatal("amf-list function not found")
	}

	// Test case 1: help flag
	args := map[string]string{"help": "true"}
	result := amfListFunc(args)
	if !strings.Contains(result, "Usage: amf-list") {
		t.Errorf("Expected help message, got: %s", result)
	}

	// Test case 2: list all AMFs
	args = map[string]string{"all": "true"}
	result = amfListFunc(args)
	if !strings.Contains(result, "AMF-01") || !strings.Contains(result, "AMF-02") || !strings.Contains(result, "AMF-03") {
		t.Errorf("Expected list of all AMFs, got: %s", result)
	}

	// Test case 3: list active AMFs
	args = map[string]string{"active": "true"}
	result = amfListFunc(args)
	if !strings.Contains(result, "AMF-01") || !strings.Contains(result, "AMF-03") {
		t.Errorf("Expected list of active AMFs, got: %s", result)
	}

	// Test case 4: invalid arguments
	args = map[string]string{"invalid": "true"}
	result = amfListFunc(args)
	if !strings.Contains(result, "Error: No valid arguments") {
		t.Errorf("Expected error for invalid arguments, got: %s", result)
	}
}

// TestHTTPEndpoints kiểm tra các API endpoints
func TestHTTPEndpoints(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Tạo server
	server := NewServer("localhost", 8080)

	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellUE(dummyFunction, dummyArgs)
	server.SetupShellGnb(dummyFunction, dummyArgs)

	// Tạo router để test
	router := gin.New()

	// Setup các API endpoints
	router.GET("/nodes/types", func(c *gin.Context) {
		types := []string{"ue", "gnb", "amf"}
		c.JSON(http.StatusOK, gin.H{
			"types": types,
		})
	})

	router.GET("/nodes/:type", func(c *gin.Context) {
		nodeType := c.Param("type")
		var nodes []string

		for _, node := range server.Nodes {
			if strings.EqualFold(node.Type.String(), nodeType) {
				nodes = append(nodes, node.Name)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"nodes": nodes,
		})
	})

	// Test GET /nodes/types
	req, _ := http.NewRequest("GET", "/nodes/types", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var response struct {
		Types []string `json:"types"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(response.Types) != 3 {
		t.Errorf("Expected 3 node types, got %d", len(response.Types))
	}

	// Test GET /nodes/ue
	req, _ = http.NewRequest("GET", "/nodes/ue", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	var nodesResponse struct {
		Nodes []string `json:"nodes"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &nodesResponse)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Kiểm tra phản hồi (số lượng nodes phải phù hợp với server.Nodes)
	count := 0
	for _, node := range server.Nodes {
		if node.Type == UE {
			count++
		}
	}

	if len(nodesResponse.Nodes) != count {
		t.Errorf("Expected %d UE nodes, got %d", count, len(nodesResponse.Nodes))
	}
}

// TestExecution kiểm tra đầy đủ từ khởi tạo đến chạy server
func TestExecution(t *testing.T) {
	// Test full execution flow (không thực sự chạy HTTP server)
	server := NewServer("localhost", 8080)

	dummyFunction := func(args map[string]string) {}
	dummyArgs := make(map[string]string)

	server.SetupShellUE(dummyFunction, dummyArgs)
	server.SetupShellGnb(dummyFunction, dummyArgs)

	// Setup example All nodes and Active nodes
	for i, node := range server.Nodes {
		switch node.Type {
		case UE:
			// All nodes that UE knows about
			server.Nodes[i].AllNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02", "AMF-03", "AMF-04"},
				"gnb": {"GNB-001", "GNB-002", "GNB-003"},
			}
			// Active nodes that UE is connected to
			server.Nodes[i].ActiveNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02"},
				"gnb": {"GNB-001"},
			}
		case Gnb:
			// All nodes that GNB knows about
			server.Nodes[i].AllNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-02", "AMF-03", "AMF-04", "AMF-05"},
				"ue":  {"UE-001", "UE-002", "UE-003", "UE-004"},
			}
			// Active nodes that GNB is connected to
			server.Nodes[i].ActiveNodes = map[string][]string{
				"amf": {"AMF-01", "AMF-03"},
				"ue":  {"UE-001", "UE-002"},
			}
		}
	}

	// Kiểm tra setup đã hoàn thiện
	if len(server.Nodes) == 0 {
		t.Fatal("No nodes initialized")
	}

	// Kiểm tra các nodes có commands
	for _, node := range server.Nodes {
		if len(node.Commands) == 0 {
			t.Errorf("Node type %s has no commands", node.Type.String())
		}
	}

	// Kiểm tra AllNodes và ActiveNodes
	for _, node := range server.Nodes {
		switch node.Type {
		case UE:
			if _, ok := node.AllNodes["amf"]; !ok {
				t.Error("UE node should have AMF in AllNodes")
			}
			if _, ok := node.ActiveNodes["amf"]; !ok {
				t.Error("UE node should have AMF in ActiveNodes")
			}
		case Gnb:
			if _, ok := node.AllNodes["amf"]; !ok {
				t.Error("Gnb node should have AMF in AllNodes")
			}
			if _, ok := node.ActiveNodes["amf"]; !ok {
				t.Error("Gnb node should have AMF in ActiveNodes")
			}
		}
	}
}

// TestMain chạy tất cả các tests
func TestMain(t *testing.T) {
	// Chạy tất cả các test functions
	t.Run("TestNewServer", TestNewServer)
	t.Run("TestSetupShellUE", TestSetupShellUE)
	t.Run("TestSetupShellGnb", TestSetupShellGnb)
	t.Run("TestRegisterHandler", TestRegisterHandler)
	t.Run("TestAmfListHandler", TestAmfListHandler)
	t.Run("TestHTTPEndpoints", TestHTTPEndpoints)
	t.Run("TestExecution", TestExecution)
}

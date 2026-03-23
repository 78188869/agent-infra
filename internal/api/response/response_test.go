package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"name": "test"}
	Success(c, data)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Code = %v, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %v, want 'success'", resp.Message)
	}
}

func TestCreated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]int{"id": 1}
	Created(c, data)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusCreated)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Code = %v, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %v, want 'success'", resp.Message)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 400 {
		t.Errorf("Code = %v, want 400", resp.Code)
	}
	if resp.Message != "invalid input" {
		t.Errorf("Message = %v, want 'invalid input'", resp.Message)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusNotFound)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 404 {
		t.Errorf("Code = %v, want 404", resp.Code)
	}
	if resp.Message != "resource not found" {
		t.Errorf("Message = %v, want 'resource not found'", resp.Message)
	}
}

func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalError(c, "database error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusInternalServerError)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 500 {
		t.Errorf("Code = %v, want 500", resp.Code)
	}
	if resp.Message != "database error" {
		t.Errorf("Message = %v, want 'database error'", resp.Message)
	}
}

func TestPaginated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []map[string]string{
		{"id": "1", "name": "item1"},
		{"id": "2", "name": "item2"},
	}

	Paginated(c, items, 100, 1, 10)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Code = %v, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %v, want 'success'", resp.Message)
	}

	// Verify paginated data structure
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("Failed to marshal data: %v", err)
	}

	var paginatedData PaginatedData
	if err := json.Unmarshal(dataBytes, &paginatedData); err != nil {
		t.Fatalf("Failed to unmarshal paginated data: %v", err)
	}

	if paginatedData.Total != 100 {
		t.Errorf("Total = %v, want 100", paginatedData.Total)
	}
	if paginatedData.Page != 1 {
		t.Errorf("Page = %v, want 1", paginatedData.Page)
	}
	if paginatedData.PageSize != 10 {
		t.Errorf("PageSize = %v, want 10", paginatedData.PageSize)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusForbidden, 403, "access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusForbidden)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 403 {
		t.Errorf("Code = %v, want 403", resp.Code)
	}
	if resp.Message != "access denied" {
		t.Errorf("Message = %v, want 'access denied'", resp.Message)
	}
}

func TestResponse_JSONFields(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]interface{}{
		"id":   "123",
		"name": "test",
	}
	Success(c, data)

	// Check that JSON has expected structure
	var raw map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := raw["code"]; !ok {
		t.Error("Response missing 'code' field")
	}
	if _, ok := raw["message"]; !ok {
		t.Error("Response missing 'message' field")
	}
	if _, ok := raw["data"]; !ok {
		t.Error("Response missing 'data' field")
	}
}

func TestPaginatedData_JSONFields(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []string{"a", "b", "c"}
	Paginated(c, items, 50, 2, 20)

	var raw map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Data field is not a map")
	}

	if _, ok := data["items"]; !ok {
		t.Error("PaginatedData missing 'items' field")
	}
	if _, ok := data["total"]; !ok {
		t.Error("PaginatedData missing 'total' field")
	}
	if _, ok := data["page"]; !ok {
		t.Error("PaginatedData missing 'page' field")
	}
	if _, ok := data["page_size"]; !ok {
		t.Error("PaginatedData missing 'page_size' field")
	}
}

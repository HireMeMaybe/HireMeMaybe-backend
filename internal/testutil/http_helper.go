package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

// MakeJSONRequest is a helper function for making JSON requests in tests
func MakeJSONRequest(body gin.H, authToken string, r *gin.Engine, endpoint string, method string) (*httptest.ResponseRecorder, map[string]interface{}) {
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(method, endpoint, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+authToken)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := map[string]interface{}{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	return rec, resp
}

func StringPtr(s string) *string {
	return &s
}
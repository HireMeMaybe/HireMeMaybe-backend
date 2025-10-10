package utilities

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

// SimulateAPICall is a helper function to simulate an API call to a gin handler function.
// It takes the handler function, route, HTTP method, and request body as parameters.
// It returns the HTTP response recorder, parsed JSON response as a map, and any error encountered. 
func SimulateAPICall(
	handlerFunc func(*gin.Context),
	route string,
	method string,
	body interface{},
) (*httptest.ResponseRecorder, map[string]interface{}, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req, err := http.NewRequest(method, route, bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	handlerFunc(c)

	var resp map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	if err != nil {
		return rec, nil, err
	}
	return rec, resp, nil
}

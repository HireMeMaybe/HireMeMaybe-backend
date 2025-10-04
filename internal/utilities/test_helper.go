package utilities

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

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

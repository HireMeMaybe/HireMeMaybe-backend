package controller

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

var testDB *database.DBinstanceStruct

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	var err error
	var midTeardown func(context.Context, ...testcontainers.TerminateOption) error
	midTeardown, testDB, err = database.GetTestDB()
	if err != nil {
		os.Exit(1)
	}
	m.Run()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if midTeardown != nil {
		_ = midTeardown(ctx)
	}
}

func makeJSONRequest(body gin.H, authToken string, r *gin.Engine, endpoint string, method string) (*httptest.ResponseRecorder, map[string]interface{}) {
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(method, endpoint, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+authToken)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := map[string]interface{}{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	return rec, resp
}

func TestGetPostByID_success(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.GET("/jobpost/:id", middleware.RequireAuth(testDB), jc.GetPostByID)

	rec, resp := makeJSONRequest(nil, userToken, r, "/jobpost/"+fmt.Sprintf("%d", database.TestJobPost1.ID), http.MethodGet)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, float64(database.TestJobPost1.ID), resp["id"])
	assert.Equal(t, database.TestJobPost1.Title, resp["title"])
}

func TestGetPostByID_notFound(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/jobpost/:id", middleware.RequireAuth(testDB), jc.GetPostByID)

	rec, _ := makeJSONRequest(nil, userToken, r, "/jobpost/999", http.MethodPost)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCreateUserReport_companyReportCpsk(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)

	body := gin.H{
		"reported_id": database.TestUserCPSK1.ID.String(),
		"reason":      "Inappropriate behavior",
	}

	rec, _ := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreateUserReport_cpskReportcpsk(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)

	body := gin.H{
		"reported_id": database.TestUserCPSK2.ID.String(),
		"reason":      "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, resp["error"], "cannot report this user")
}

func TestCreateUserReport_NotEnoughInfo(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)

	body := gin.H{
		"reason": "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "Invalid request body")
}

func TestCreateUserReport_NotFoundUser(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)

	body := gin.H{
		"reported_id": "00000000-0000-0000-0000-000000000000",
		"reason":      "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, resp["error"], "Reported user not found")
}

func TestCreateUserReport_InvalidUUID(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)

	body := gin.H{
		"reported_id": "invalid-uuid",
		"reason":      "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "Invalid request body")
}

func TestCreateUserReport_reportAdmin(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)

	body := gin.H{
		"reported_id": database.TestAdminUser.ID.String(),
		"reason":      "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, resp["error"], "cannot report this user")
}

func TestCreatePostReport_reportSuccess(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report/post", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), jc.CreatePostReport)

	body := gin.H{
		"reported_id": database.TestJobPost1.ID,
		"reason":      "Inappropriate content",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report/post", http.MethodPost)

	log.Println(resp["error"])
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestCreatePostReport_postNotFound(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report/post", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), jc.CreatePostReport)

	body := gin.H{
		"reported_id": 9999,
		"reason":      "Inappropriate content",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report/post", http.MethodPost)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, resp["error"], "Reported post not found")
}

func TestCreatePostReport_invalidRequestBody(t *testing.T) {
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report/post", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), jc.CreatePostReport)

	body := gin.H{
		"reason": "Inappropriate content",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report/post", http.MethodPost)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "Invalid request body")
}

func TestUpdateReportStatus_ResolvedUser(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// First, create a report to update
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)
	r.PUT("/report/:type/:id", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.UpdateReportStatus)

	body := gin.H{
		"reported_id": database.TestUserCompany1.ID.String(),
		"reason":      "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)
	assert.Equal(t, http.StatusCreated, rec.Code)

	floatID, _ := resp["report_id"].(float64)

	reportID, _ := strconv.Atoi(strconv.FormatInt(int64(floatID), 10))
	// Now update the report status
	updateBody := gin.H{
		"status":     "resolved",
		"admin_note": "Reviewed and resolved",
	}

	rec, updateResp := makeJSONRequest(updateBody, adminToken, r, "/report/user/"+strconv.Itoa(reportID), http.MethodPut)

	assert.Equal(t, http.StatusOK, rec.Code)
	log.Println(updateResp["error"])
	assert.Contains(t, updateResp["message"], "Report status updated successfully")
}

func TestUpdateReportStatus_ResolvedPost(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// First, create a report to update
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report/post", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), jc.CreatePostReport)
	r.PUT("/report/:type/:id", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.UpdateReportStatus)

	body := gin.H{
		"reported_id": database.TestJobPost1.ID,
		"reason":      "Inappropriate content",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report/post", http.MethodPost)
	assert.Equal(t, http.StatusCreated, rec.Code)

	floatID, _ := resp["report_id"].(float64)

	reportID, _ := strconv.Atoi(strconv.FormatInt(int64(floatID), 10))
	// Now update the report status
	updateBody := gin.H{
		"status":     "resolved",
		"admin_note": "Reviewed and resolved",
	}

	rec, updateResp := makeJSONRequest(updateBody, adminToken, r, "/report/post/"+strconv.Itoa(reportID), http.MethodPut)

	assert.Equal(t, http.StatusOK, rec.Code)
	log.Println(updateResp["error"])
	assert.Contains(t, updateResp["message"], "Report status updated successfully")
}

func TestUpdateReportStatus_InvalidStatus(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// First, create a report to update
	reporterToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.POST("/report", middleware.RequireAuth(testDB), jc.CreateUserReport)
	r.PUT("/report/:type/:id", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.UpdateReportStatus)

	body := gin.H{
		"reported_id": database.TestUserCompany1.ID.String(),
		"reason":      "Inappropriate behavior",
	}

	rec, resp := makeJSONRequest(body, reporterToken, r, "/report", http.MethodPost)
	assert.Equal(t, http.StatusCreated, rec.Code)

	floatID, _ := resp["report_id"].(float64)

	reportID, _ := strconv.Atoi(strconv.FormatInt(int64(floatID), 10))
	// Now update the report status with invalid status
	updateBody := gin.H{
		"status":     "invalid_status",
		"admin_note": "Reviewed and resolved",
	}

	rec, updateResp := makeJSONRequest(updateBody, adminToken, r, "/report/user/"+strconv.Itoa(reportID), http.MethodPut)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, updateResp["error"], "Invalid request body")
}

func TestUpdateReportStatus_ReportNotFound(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.PUT("/report/:type/:id", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.UpdateReportStatus)

	// Update a non-existent report
	updateBody := gin.H{
		"status":     "resolved",
		"admin_note": "Reviewed and resolved",
	}

	rec, updateResp := makeJSONRequest(updateBody, adminToken, r, "/report/user/9999", http.MethodPut)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, updateResp["error"], "Report not found")
}

func TestUpdateReportStatus_InvalidReportType(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobController{
		DB: testDB,
	}
	r.PUT("/report/:type/:id", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.UpdateReportStatus)

	// Update a report with invalid type
	updateBody := gin.H{
		"status":     "resolved",
		"admin_note": "Reviewed and resolved",
	}

	rec, updateResp := makeJSONRequest(updateBody, adminToken, r, "/report/invalid_type/1", http.MethodPut)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, updateResp["error"], "Invalid report type")
}

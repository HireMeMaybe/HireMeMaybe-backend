package company

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/testutil"
	"context"
	"net/http"
	"os"
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

func TestEditCompanyProfile_NonCompanyForbidden(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole("company"), cc.EditCompanyProfile)

	body := gin.H{"name": "Malicious Update"}

	rec, resp := testutil.MakeJSONRequest(body, cpskToken, r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "permission")
	}
}

func TestEditCompanyProfile_Success(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), cc.EditCompanyProfile)

	// Update company profile with new information
	body := gin.H{
		"name":     "Updated Company Name",
		"overview": "This is an updated company overview",
		"industry": "Updated Technology",
	}

	rec, resp := testutil.MakeJSONRequest(body, companyToken, r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, resp)
	assert.Equal(t, "Updated Company Name", resp["name"])
	assert.Equal(t, "This is an updated company overview", resp["overview"])
	assert.Equal(t, "Updated Technology", resp["industry"])
}

func TestEditCompanyProfile_Unauthorized(t *testing.T) {
	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), cc.EditCompanyProfile)

	body := gin.H{"name": "Unauthorized Update"}

	rec, resp := testutil.MakeJSONRequest(body, "", r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "authorization header")
}

func TestEditCompanyProfile_PartialUpdate(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), cc.EditCompanyProfile)

	// Only update the industry field
	body := gin.H{
		"industry": "Software Development",
	}

	rec, resp := testutil.MakeJSONRequest(body, companyToken, r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, resp)
	assert.Equal(t, "Software Development", resp["industry"])
}

func TestEditCompanyProfile_UpdateUserInfo(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), cc.EditCompanyProfile)

	// Update both company and user info
	body := gin.H{
		"name": "New Company Name",
		"tel":  "+1-555-9999",
	}

	rec, resp := testutil.MakeJSONRequest(body, companyToken, r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, resp)
	assert.Equal(t, "New Company Name", resp["name"])

	// Verify user info was updated
	userInfo, ok := resp["User"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "+1-555-9999", userInfo["tel"])
}

func TestEditCompanyProfile_InvalidJSON(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), cc.EditCompanyProfile)

	// Send invalid JSON with unknown field (should fail due to DisallowUnknownFields)
	body := gin.H{
		"name":          "Valid Name",
		"unknown_field": "This should cause an error",
	}

	rec, resp := testutil.MakeJSONRequest(body, companyToken, r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "Invalid request body")
}

func TestEditCompanyProfile_EmptyBody(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.PATCH("/company/profile", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), cc.EditCompanyProfile)

	// Send empty body - should succeed as MergeNonEmpty won't overwrite with empty values
	body := gin.H{}

	rec, _ := testutil.MakeJSONRequest(body, companyToken, r, "/company/profile", http.MethodPatch)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCompanyByID_Success(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.GET("/company/:company_id", middleware.RequireAuth(testDB), cc.GetCompanyByID)

	rec, resp := testutil.MakeJSONRequest(nil, userToken, r, "/company/"+database.TestCompany1.UserID.String(), http.MethodGet)

	assert.Equal(t, http.StatusOK, rec.Code)
	if resp != nil {
		// ensure some expected field exists
		_, ok := resp["id"]
		assert.True(t, ok)
	}
}

func TestGetMyCompanyProfile_Success(t *testing.T) {
	// use company user token to fetch own profile
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CompanyController{DB: testDB}
	r.GET("/company/myprofile", middleware.RequireAuth(testDB), cc.GetMyCompanyProfile)

	rec, resp := testutil.MakeJSONRequest(nil, companyToken, r, "/company/myprofile", http.MethodGet)

	assert.Equal(t, http.StatusOK, rec.Code)
	if resp != nil {
		_, ok := resp["id"]
		assert.True(t, ok)
	}
}

package company

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
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

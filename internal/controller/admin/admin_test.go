package admin

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"net/http"
	"os"
	"testing"
	"time"
	"HireMeMaybe-backend/internal/testutil"
	"HireMeMaybe-backend/internal/model"
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

func TestGetCompanies(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-companies", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCompanies)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-companies", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCPSK(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCompanies)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-cpsk", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)
}

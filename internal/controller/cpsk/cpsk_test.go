package cpsk

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

func TestEditCPSKProfile_UnknownFieldBadRequest(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CPSKController{DB: testDB}
	r.PATCH("/cpsk/profile", middleware.RequireAuth(testDB), middleware.CheckRole("cpsk"), cc.EditCPSKProfile)

	body := gin.H{"first_name": "NewName", "unknown_field": "x"}

	rec, resp := testutil.MakeJSONRequest(body, cpskToken, r, "/cpsk/profile", http.MethodPatch)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "Invalid request body")
	}
}

func TestGetMyCPSKProfile_Success(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	cc := &CPSKController{DB: testDB}
	r.GET("/cpsk/myprofile", middleware.RequireAuth(testDB), middleware.CheckRole("cpsk"), cc.GetMyCPSKProfile)

	rec, resp := testutil.MakeJSONRequest(nil, cpskToken, r, "/cpsk/myprofile", http.MethodGet)

	assert.Equal(t, http.StatusOK, rec.Code)
	if resp != nil {
		_, ok := resp["user_id"]
		assert.True(t, ok)
	}
}

package jobpost

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/testutil"
	"context"
	"fmt"
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

func TestGetPostByID_success(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobPostController{
		DB: testDB,
	}
	r.GET("/jobpost/:id", middleware.RequireAuth(testDB), jc.GetPostByID)

	rec, resp := testutil.MakeJSONRequest(nil, userToken, r, "/jobpost/"+fmt.Sprintf("%d", database.TestJobPost1.ID), http.MethodGet)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, float64(database.TestJobPost1.ID), resp["id"])
	assert.Equal(t, database.TestJobPost1.Title, resp["title"])
}

func TestGetPostByID_notFound(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	r := gin.Default()
	jc := &JobPostController{
		DB: testDB,
	}
	r.POST("/jobpost/:id", middleware.RequireAuth(testDB), jc.GetPostByID)

	rec, _ := testutil.MakeJSONRequest(nil, userToken, r, "/jobpost/999", http.MethodPost)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

package application

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

func TestApplicationHandler_Success(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// create a resume file record to reference
	f := model.File{Content: []byte("resume"), Extension: ".pdf"}
	if err := testDB.Create(&f).Error; err != nil {
		t.Fatalf("failed to create resume file: %v", err)
	}

	r := gin.Default()
	ac := &ApplicationController{DB: testDB}
	r.POST("/application", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), ac.ApplicationHandler)

	body := gin.H{
		"post_id":   database.TestJobPost1.ID,
		"resume_id": f.ID,
	}

	rec, resp := testutil.MakeJSONRequest(body, cpskToken, r, "/application", http.MethodPost)

	assert.Equal(t, http.StatusCreated, rec.Code)
	// response should contain post_id and cpsk_id fields
	if resp != nil {
		v, ok := resp["post_id"]
		assert.True(t, ok)
		assert.Equal(t, float64(database.TestJobPost1.ID), v)
	}
}

func TestApplicationHandler_Duplicate(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// ensure a resume file exists
	f := model.File{Content: []byte("resume2"), Extension: ".pdf"}
	if err := testDB.Create(&f).Error; err != nil {
		t.Fatalf("failed to create resume file: %v", err)
	}

	// Clean up any existing application for this CPSK and post to ensure test isolation
	if err := testDB.Where("post_id = ? AND cpsk_id = ?", database.TestJobPost1.ID, database.TestUserCPSK1.ID).
		Delete(&model.Application{}).Error; err != nil {
		t.Fatalf("failed to cleanup existing application: %v", err)
	}

	r := gin.Default()
	ac := &ApplicationController{DB: testDB}
	r.POST("/application", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), ac.ApplicationHandler)

	body := gin.H{"post_id": database.TestJobPost1.ID, "resume_id": f.ID}

	rec, _ := testutil.MakeJSONRequest(body, cpskToken, r, "/application", http.MethodPost)
	// first attempt may succeed
	if rec.Code == http.StatusCreated {
		// second attempt should be duplicate
		rec2, resp2 := testutil.MakeJSONRequest(body, cpskToken, r, "/application", http.MethodPost)
		assert.Equal(t, http.StatusBadRequest, rec2.Code)
		if resp2 != nil {
			assert.Contains(t, resp2["error"], "already applied")
		}
	} else {
		t.Fatalf("initial application failed with code %d", rec.Code)
	}
}

func TestApplicationHandler_InvalidPostID(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// create a resume file record to reference
	f := model.File{Content: []byte("resume3"), Extension: ".pdf"}
	if err := testDB.Create(&f).Error; err != nil {
		t.Fatalf("failed to create resume file: %v", err)
	}

	r := gin.Default()
	ac := &ApplicationController{DB: testDB}
	r.POST("/application", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCPSK), ac.ApplicationHandler)

	body := gin.H{"post_id": 999999, "resume_id": f.ID}

	rec, resp := testutil.MakeJSONRequest(body, cpskToken, r, "/application", http.MethodPost)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "job post not found")
	}
}

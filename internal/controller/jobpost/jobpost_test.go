package jobpost

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/testutil"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"

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

func TestGetPostByID_BannedCompanyNotFound(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// load the job post and its company user
	var post model.JobPost
	if err := testDB.Preload("CompanyUser.User").Where("id = ?", database.TestJobPost1.ID).First(&post).Error; err != nil {
		t.Fatalf("failed to load job post: %v", err)
	}

	// attach a permanent ban to the company user
	var companyUser model.User
	if err := testDB.Where("id = ?", post.CompanyUser.UserID).First(&companyUser).Error; err != nil {
		t.Fatalf("failed to load company user: %v", err)
	}
	now := time.Now()
	punishment := model.PunishmentStruct{PunishmentType: model.BanPunishment, PunishAt: &now, PunishEnd: nil}
	companyUser.Punishment = &punishment
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&companyUser).Error; err != nil {
		t.Fatalf("failed to save punishment: %v", err)
	}

	// request the job post as a CPSK user
	r := gin.Default()
	jc := &JobPostController{DB: testDB}
	r.GET("/jobpost/:id", middleware.RequireAuth(testDB), jc.GetPostByID)

	rec, _ := testutil.MakeJSONRequest(nil, userToken, r, "/jobpost/"+fmt.Sprintf("%d", database.TestJobPost1.ID), http.MethodGet)

	// banned company's post should be treated as not found
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// cleanup punishment
	companyUser.Punishment = nil
	companyUser.PunishmentID = nil
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&companyUser).Error; err != nil {
		t.Fatalf("failed to cleanup punishment: %v", err)
	}
}

func TestGetPosts_ReturnArray(t *testing.T) {
	userToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobPostController{DB: testDB}
	r.GET("/jobpost", middleware.RequireAuth(testDB), jc.GetPosts)

	rec, _ := testutil.MakeJSONRequest(nil, userToken, r, "/jobpost", http.MethodGet)

	assert.Equal(t, http.StatusOK, rec.Code)

	var posts []map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &posts))
	assert.GreaterOrEqual(t, len(posts), 1)
}

func TestCreateJobPost_Success(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &JobPostController{DB: testDB}
	r.POST("/jobpost", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), jc.CreateJobPostHandler)

	body := gin.H{
		"title":        "New Internship",
		"desc":         "Work on APIs",
		"req":          "Go",
		"exp_lvl":      "Internship",
		"location":     "Remote",
		"type":         "Internship",
		"salary":       "0",
		"tags":         []string{"go"},
		"default_form": true,
	}

	rec, resp := testutil.MakeJSONRequest(body, companyToken, r, "/jobpost", http.MethodPost)
	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	if resp != nil {
		assert.Equal(t, "New Internship", resp["title"])
	}
}

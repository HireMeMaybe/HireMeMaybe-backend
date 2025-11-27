package admin

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/middleware"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/testutil"
	"context"
	"encoding/json"
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
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCPSK)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-cpsk", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Response should be an array
	var cpskList []model.CPSKUser
	err = json.Unmarshal(rec.Body.Bytes(), &cpskList)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(cpskList), 0)
}

func TestGetCPSK_Unauthorized(t *testing.T) {
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCPSK)
	rec, resp := testutil.MakeJSONRequest(nil, "", r, "/get-cpsk", http.MethodGet)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "authorization header")
}

func TestGetCPSK_WrongRole(t *testing.T) {
	// Login as CPSK user (not admin)
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCPSK)
	rec, resp := testutil.MakeJSONRequest(nil, cpskToken, r, "/get-cpsk", http.MethodGet)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, resp["error"], "permission")
}

func TestGetCPSK_WithPunishmentFilter(t *testing.T) {
	// Create a CPSK user with ban punishment
	banEnd := time.Now().Add(48 * time.Hour)
	punishment := model.PunishmentStruct{
		PunishmentType: "ban",
		PunishEnd:      &banEnd,
	}
	if err := testDB.Create(&punishment).Error; err != nil {
		t.Fatal(err)
	}

	punishmentID := int(punishment.ID)
	testUser := model.User{
		Username:     "test_banned_cpsk",
		Password:     "hashed_password",
		Role:         model.RoleCPSK,
		PunishmentID: &punishmentID,
	}
	if err := testDB.Create(&testUser).Error; err != nil {
		t.Fatal(err)
	}

	cpskUser := model.CPSKUser{
		UserID: testUser.ID,
		EditableCPSKInfo: model.EditableCPSKInfo{
			FirstName: "Banned",
			LastName:  "User",
		},
	}
	if err := testDB.Create(&cpskUser).Error; err != nil {
		t.Fatal(err)
	}

	// Test with ban filter
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCPSK)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-cpsk?punishment=ban", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)

	var cpskList []model.CPSKUser
	err = json.Unmarshal(rec.Body.Bytes(), &cpskList)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(cpskList), 1, "Should return at least the banned user")

	// Cleanup
	testDB.Unscoped().Delete(&cpskUser)
	testDB.Unscoped().Delete(&testUser)
	testDB.Unscoped().Delete(&punishment)
}

func TestGetCPSK_WithMultiplePunishmentFilters(t *testing.T) {
	// Create users with different punishment types
	banPunishment := model.PunishmentStruct{
		PunishmentType: "ban",
		PunishEnd:      nil, // Permanent
	}
	if err := testDB.Create(&banPunishment).Error; err != nil {
		t.Fatal(err)
	}

	suspendEnd := time.Now().Add(24 * time.Hour)
	suspendPunishment := model.PunishmentStruct{
		PunishmentType: "suspend",
		PunishEnd:      &suspendEnd,
	}
	if err := testDB.Create(&suspendPunishment).Error; err != nil {
		t.Fatal(err)
	}

	banID := int(banPunishment.ID)
	bannedUser := model.User{
		Username:     "test_banned_cpsk2",
		Password:     "hashed_password",
		Role:         model.RoleCPSK,
		PunishmentID: &banID,
	}
	if err := testDB.Create(&bannedUser).Error; err != nil {
		t.Fatal(err)
	}

	bannedCPSK := model.CPSKUser{
		UserID: bannedUser.ID,
		EditableCPSKInfo: model.EditableCPSKInfo{
			FirstName: "Banned",
			LastName:  "CPSK",
		},
	}
	if err := testDB.Create(&bannedCPSK).Error; err != nil {
		t.Fatal(err)
	}

	suspendID := int(suspendPunishment.ID)
	suspendedUser := model.User{
		Username:     "test_suspended_cpsk",
		Password:     "hashed_password",
		Role:         model.RoleCPSK,
		PunishmentID: &suspendID,
	}
	if err := testDB.Create(&suspendedUser).Error; err != nil {
		t.Fatal(err)
	}

	suspendedCPSK := model.CPSKUser{
		UserID: suspendedUser.ID,
		EditableCPSKInfo: model.EditableCPSKInfo{
			FirstName: "Suspended",
			LastName:  "CPSK",
		},
	}
	if err := testDB.Create(&suspendedCPSK).Error; err != nil {
		t.Fatal(err)
	}

	// Test with both ban and suspend filter
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCPSK)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-cpsk?punishment=ban+suspend", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)

	var cpskList []model.CPSKUser
	err = json.Unmarshal(rec.Body.Bytes(), &cpskList)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(cpskList), 2, "Should return both banned and suspended users")

	// Cleanup
	testDB.Unscoped().Delete(&bannedCPSK)
	testDB.Unscoped().Delete(&suspendedCPSK)
	testDB.Unscoped().Delete(&bannedUser)
	testDB.Unscoped().Delete(&suspendedUser)
	testDB.Unscoped().Delete(&banPunishment)
	testDB.Unscoped().Delete(&suspendPunishment)
}

func TestGetCPSK_NoPunishmentFilter(t *testing.T) {
	// Test without punishment filter - should return all CPSK users
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-cpsk", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetCPSK)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-cpsk", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)

	var cpskList []model.CPSKUser
	err = json.Unmarshal(rec.Body.Bytes(), &cpskList)
	assert.NoError(t, err)
	// Should return all CPSK users from test seed
	assert.GreaterOrEqual(t, len(cpskList), 1)
}

func TestGetVisitors(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-visitors", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetVisitors)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-visitors", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var visitorList []model.VisitorUser
	err = json.Unmarshal(rec.Body.Bytes(), &visitorList)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(visitorList), 0)
}

func TestGetVisitors_Unauthorized(t *testing.T) {
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-visitors", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetVisitors)
	rec, resp := testutil.MakeJSONRequest(nil, "", r, "/get-visitors", http.MethodGet)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "authorization header")
}

func TestGetVisitors_WrongRole(t *testing.T) {
	// Login as CPSK user (not admin)
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-visitors", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetVisitors)
	rec, resp := testutil.MakeJSONRequest(nil, cpskToken, r, "/get-visitors", http.MethodGet)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, resp["error"], "permission")
}

func TestGetVisitors_WithPunishmentFilter(t *testing.T) {
	// Create a visitor user with ban punishment
	banEnd := time.Now().Add(48 * time.Hour)
	punishment := model.PunishmentStruct{
		PunishmentType: "ban",
		PunishEnd:      &banEnd,
	}
	if err := testDB.Create(&punishment).Error; err != nil {
		t.Fatal(err)
	}

	punishmentID := int(punishment.ID)
	testUser := model.User{
		Username:     "test_banned_visitor",
		Password:     "hashed_password",
		Role:         model.RoleVisitor,
		PunishmentID: &punishmentID,
	}
	if err := testDB.Create(&testUser).Error; err != nil {
		t.Fatal(err)
	}

	visitorUser := model.VisitorUser{
		UserID: testUser.ID,
		EditableVisitorInfo: model.EditableVisitorInfo{
			FirstName: "Banned",
			LastName:  "Visitor",
		},
	}
	if err := testDB.Create(&visitorUser).Error; err != nil {
		t.Fatal(err)
	}

	// Test with ban filter
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-visitors", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetVisitors)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-visitors?punishment=ban", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var visitorList []model.VisitorUser
	err = json.Unmarshal(rec.Body.Bytes(), &visitorList)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(visitorList), 1, "Should return at least the banned visitor")

	// Cleanup
	testDB.Unscoped().Delete(&visitorUser)
	testDB.Unscoped().Delete(&testUser)
	testDB.Unscoped().Delete(&punishment)
}

func TestGetVisitors_WithMultiplePunishmentFilters(t *testing.T) {
	// Create visitors with different punishment types
	banPunishment := model.PunishmentStruct{
		PunishmentType: "ban",
		PunishEnd:      nil, // Permanent
	}
	if err := testDB.Create(&banPunishment).Error; err != nil {
		t.Fatal(err)
	}

	suspendEnd := time.Now().Add(24 * time.Hour)
	suspendPunishment := model.PunishmentStruct{
		PunishmentType: "suspend",
		PunishEnd:      &suspendEnd,
	}
	if err := testDB.Create(&suspendPunishment).Error; err != nil {
		t.Fatal(err)
	}

	banID := int(banPunishment.ID)
	bannedUser := model.User{
		Username:     "test_banned_visitor2",
		Password:     "hashed_password",
		Role:         model.RoleVisitor,
		PunishmentID: &banID,
	}
	if err := testDB.Create(&bannedUser).Error; err != nil {
		t.Fatal(err)
	}

	bannedVisitor := model.VisitorUser{
		UserID: bannedUser.ID,
		EditableVisitorInfo: model.EditableVisitorInfo{
			FirstName: "Banned",
			LastName:  "Visitor",
		},
	}
	if err := testDB.Create(&bannedVisitor).Error; err != nil {
		t.Fatal(err)
	}

	suspendID := int(suspendPunishment.ID)
	suspendedUser := model.User{
		Username:     "test_suspended_visitor",
		Password:     "hashed_password",
		Role:         model.RoleVisitor,
		PunishmentID: &suspendID,
	}
	if err := testDB.Create(&suspendedUser).Error; err != nil {
		t.Fatal(err)
	}

	suspendedVisitor := model.VisitorUser{
		UserID: suspendedUser.ID,
		EditableVisitorInfo: model.EditableVisitorInfo{
			FirstName: "Suspended",
			LastName:  "Visitor",
		},
	}
	if err := testDB.Create(&suspendedVisitor).Error; err != nil {
		t.Fatal(err)
	}

	// Test with both ban and suspend filter
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-visitors", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetVisitors)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-visitors?punishment=ban+suspend", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var visitorList []model.VisitorUser
	err = json.Unmarshal(rec.Body.Bytes(), &visitorList)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(visitorList), 2, "Should return both banned and suspended visitors")

	// Cleanup
	testDB.Unscoped().Delete(&bannedVisitor)
	testDB.Unscoped().Delete(&suspendedVisitor)
	testDB.Unscoped().Delete(&bannedUser)
	testDB.Unscoped().Delete(&suspendedUser)
	testDB.Unscoped().Delete(&banPunishment)
	testDB.Unscoped().Delete(&suspendPunishment)
}

func TestGetVisitors_NoPunishmentFilter(t *testing.T) {
	// Test without punishment filter - should return all visitor users
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)
	
	r := gin.Default()
	jc := &AdminController{
		DB: testDB,
	}
	r.GET("/get-visitors", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleAdmin), jc.GetVisitors)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/get-visitors", http.MethodGet)
	assert.Equal(t, http.StatusOK, rec.Code)
	
	var visitorList []model.VisitorUser
	err = json.Unmarshal(rec.Body.Bytes(), &visitorList)
	assert.NoError(t, err)
	// Should return all visitor users from test seed
	assert.GreaterOrEqual(t, len(visitorList), 0)
}

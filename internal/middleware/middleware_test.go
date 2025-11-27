package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/controller/application"
	"HireMeMaybe-backend/internal/controller/jobpost"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/testutil"
	"HireMeMaybe-backend/internal/utilities"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"gorm.io/gorm"
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

func protectedEngine() *gin.Engine {
	r := gin.New()
	r.GET("/protected", RequireAuth(testDB), checkUserHandler)
	return r
}

func checkUserHandler(c *gin.Context) {
	u, exist := c.Get("user")
	if !exist {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "user": u})
}

func readFileHandler(c *gin.Context) {
	rawFile, err := c.FormFile("file")
	if err != nil {

		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Entity too large",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve file: %s", err.Error()),
		})
		return
	}

	f, err := rawFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot open file", "ok": false})
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("Failed to close file")
		}
	}()

	stat := rawFile.Size
	log.Println("File size:", stat)

	if _, err := io.ReadAll(f); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read file", "ok": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func getCheckRoleHandler(role ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, exist := c.Get("user")
		if !exist {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false})
			return
		}
		user, err := utilities.ExtractUser(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": err.Error()})
			return
		}
		if !utilities.Contains(role, user.Role) {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "User doesn't have permission to access"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "user": u, "message": "Hello, " + user.Role})
	}
}

func convertFileToBytes(filePath string) ([]byte, error) {
	// #nosec G304 -- test inputs are static files under ./testfile checked into repo
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatal("Failed to close file")
		}
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := fileInfo.Size()
	fileBytes := make([]byte, fileSize)

	_, err = file.Read(fileBytes)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func simulateFileSendingRequest(t *testing.T, engine *gin.Engine, endpoint string, filePath string, fileKey string) *httptest.ResponseRecorder {
	fileBytes, err := convertFileToBytes(filePath)
	assert.NoError(t, err)

	bodyReader, bodyWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(bodyWriter)

	go func() {
		part, err := multipartWriter.CreateFormFile(fileKey, filePath)
		assert.NoError(t, err)
		_, err = part.Write(fileBytes)
		assert.NoError(t, err)
		if err := multipartWriter.Close(); err != nil {
			log.Fatal("Failed to close multipart writer")
		}
		if err := bodyWriter.Close(); err != nil {
			log.Fatal("Failed to close body writer")
		}
	}()

	req, _ := http.NewRequest(http.MethodPost, endpoint, bodyReader)
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestRequireAuth_Success(t *testing.T) {
	engine := gin.New()
	engine.GET("/protected", RequireAuth(testDB), checkUserHandler)
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
}

func TestRequireAuth_NoHeader(t *testing.T) {
	engine := protectedEngine()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "Invalid authorization header")
}

func TestRequireAuth_ExpiredToken(t *testing.T) {
	engine := protectedEngine()
	token, _, err := auth.GenerateTokenWithDuration(database.TestUserCPSK1.ID, -1*time.Minute, auth.JwtIssuer)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Equal(t, "Access token expired", body["error"])
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	engine := protectedEngine()
	// Create a valid token then corrupt it (signature mismatch)
	validToken, _, err := auth.GenerateTokenWithDuration(database.TestUserCPSK1.ID, time.Hour, auth.JwtIssuer)
	assert.NoError(t, err)
	invalid := validToken + "x"

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+invalid)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "Failed to validate token")
}

func TestRequireAuth_UnknownUser(t *testing.T) {
	engine := protectedEngine()
	randomID := uuid.New()
	token, _, err := auth.GenerateTokenWithDuration(randomID, time.Hour, auth.JwtIssuer)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// Current middleware reports DB retrieval error (500)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "User not exist")

}

func TestRequireAuth_InvalidIssuer(t *testing.T) {
	engine := protectedEngine()
	token, _, err := auth.GenerateTokenWithDuration(database.TestCPSK1.UserID, time.Hour, "invalid-issuer")
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "Invalid token issuer")
}

func TestCheckRole_NoRequireAuthBefore(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", CheckRole(model.RoleCPSK), getCheckRoleHandler("cpsk"))
	req, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.Contains(t, body["error"], "user information not provided")
}

func TestCheckRole_WrongRole(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", RequireAuth(testDB), CheckRole(model.RoleCompany), getCheckRoleHandler("company"))
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "User doesn't have permission to access")
}

func TestCheckRole_Success(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", RequireAuth(testDB), CheckRole(model.RoleCPSK), getCheckRoleHandler("cpsk"))
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["message"], "Hello, cpsk")
}

func TestCheckRole_MultipleRoleCheck(t *testing.T) {
	engine := gin.New()
	engine.GET("/need-role", RequireAuth(testDB), CheckRole(model.RoleCPSK, model.RoleAdmin), getCheckRoleHandler(model.RoleCPSK, model.RoleAdmin))

	// Test with CPSK user
	tokenCPSK, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	reqCPSK, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	reqCPSK.Header.Set("Authorization", "Bearer "+tokenCPSK)
	recCPSK := httptest.NewRecorder()
	engine.ServeHTTP(recCPSK, reqCPSK)

	assert.Equal(t, http.StatusOK, recCPSK.Code)
	var bodyCPSK map[string]interface{}
	assert.NoError(t, json.Unmarshal(recCPSK.Body.Bytes(), &bodyCPSK))
	assert.Contains(t, bodyCPSK["message"], "Hello, cpsk")

	// Test with Admin user
	tokenAdmin, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	reqAdmin, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+tokenAdmin)
	recAdmin := httptest.NewRecorder()
	engine.ServeHTTP(recAdmin, reqAdmin)

	assert.Equal(t, http.StatusOK, recAdmin.Code)
	var bodyAdmin map[string]interface{}
	assert.NoError(t, json.Unmarshal(recAdmin.Body.Bytes(), &bodyAdmin))
	assert.Contains(t, bodyAdmin["message"], "Hello, admin")

	// Test with Company user (should be forbidden)
	tokenCompany, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	reqCompany, _ := http.NewRequest(http.MethodGet, "/need-role", nil)
	reqCompany.Header.Set("Authorization", "Bearer "+tokenCompany)
	recCompany := httptest.NewRecorder()
	engine.ServeHTTP(recCompany, reqCompany)

	assert.Equal(t, http.StatusForbidden, recCompany.Code)
	var bodyCompany map[string]interface{}
	assert.NoError(t, json.Unmarshal(recCompany.Body.Bytes(), &bodyCompany))
	assert.Contains(t, bodyCompany["error"], "User doesn't have permission to access")
}

func TestSizeLimit_LessThenLimit(t *testing.T) {
	engine := gin.New()
	engine.POST("/upload", SizeLimit(10<<20), readFileHandler)

	rec := simulateFileSendingRequest(t, engine, "/upload", "./testfile/test1mb.jpeg", "file")

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
}

func TestSizeLimit_EqualLimit(t *testing.T) {
	engine := gin.New()
	engine.POST("/upload", SizeLimit(10<<20), readFileHandler)

	rec := simulateFileSendingRequest(t, engine, "/upload", "./testfile/test10mb.jpeg", "file")

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
}

func TestSizeLimit_ExceedLimid(t *testing.T) {
	engine := gin.New()
	engine.POST("/upload", SizeLimit(10<<20), readFileHandler)

	rec := simulateFileSendingRequest(t, engine, "/upload", "./testfile/test11mb.jpeg", "file")

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "Entity too large")
}

func TestSizeLimit_WayExceedLimit(t *testing.T) {
	engine := gin.New()
	engine.POST("/upload", SizeLimit(10<<20), readFileHandler)

	rec := simulateFileSendingRequest(t, engine, "/upload", "./testfile/test100mb.jpeg", "file")

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "Entity too large")
}

func TestCheckPunishment_SuspendedCompanyCannotPost(t *testing.T) {
	companyToken, err := auth.GetAccessToken(t, testDB, database.TestUserCompany1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// attach a suspend punishment to the company user
	var user model.User
	if err := testDB.Where("id = ?", database.TestUserCompany1.ID).First(&user).Error; err != nil {
		t.Fatalf("failed to load company user: %v", err)
	}
	now := time.Now()
	future := now.Add(48 * time.Hour)
	punishment := model.PunishmentStruct{PunishmentType: model.SuspendPunishment, PunishAt: &now, PunishEnd: &future}
	user.Punishment = &punishment
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("failed to save punishment: %v", err)
	}

	engine := gin.New()
	jc := &jobpost.JobPostController{DB: testDB}
	engine.POST("/jobpost", RequireAuth(testDB), CheckRole(model.RoleCompany), CheckPunishment(testDB, model.SuspendPunishment), jc.CreateJobPostHandler)

	body := gin.H{
		"title":        "Suspended Company Posting",
		"desc":         "Should be blocked",
		"req":          "None",
		"exp_lvl":      "Intern",
		"location":     "Remote",
		"type":         "Intern",
		"salary":       "0",
		"tags":         []string{"go"},
		"default_form": true,
	}

	rec, resp := testutil.MakeJSONRequest(body, companyToken, engine, "/jobpost", http.MethodPost)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "don't have access")
	}

	// cleanup punishment
	user.Punishment = nil
	user.PunishmentID = nil
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("failed to cleanup punishment: %v", err)
	}
}

func TestCheckPunishment_SuspendedCPSKCannotApply(t *testing.T) {
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// create a resume file record to reference
	f := model.File{Content: []byte("resume-ban"), Extension: ".pdf"}
	if err := testDB.Create(&f).Error; err != nil {
		t.Fatalf("failed to create resume file: %v", err)
	}

	// Attach a ban punishment to the CPSK user (unexpired)
	var user model.User
	if err := testDB.Where("id = ?", database.TestUserCPSK1.ID).First(&user).Error; err != nil {
		t.Fatalf("failed to load user: %v", err)
	}
	now := time.Now()
	future := now.Add(24 * time.Hour)
	punishment := model.PunishmentStruct{PunishmentType: model.SuspendPunishment, PunishAt: &now, PunishEnd: &future}
	user.Punishment = &punishment
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("failed to save punishment: %v", err)
	}

	// Attempt to apply - should be blocked by middleware
	engine := gin.New()
	ac := &application.ApplicationController{DB: testDB}
	engine.POST("/application", RequireAuth(testDB), CheckPunishment(testDB, model.SuspendPunishment), CheckRole(model.RoleCPSK), ac.ApplicationHandler)

	body := gin.H{"post_id": database.TestJobPost1.ID, "resume_id": f.ID}

	rec, resp := testutil.MakeJSONRequest(body, cpskToken, engine, "/application", http.MethodPost)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	if resp != nil {
		// message should indicate lack of access
		assert.Contains(t, resp["error"], "don't have access")
	}

	// Clean up: remove punishment so other tests unaffected
	user.Punishment = nil
	user.PunishmentID = nil
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("failed to cleanup punishment: %v", err)
	}
}

func TestCheckPunishment_BannedUserCannotAccess(t *testing.T) {
	// Use CPSK user for this test
	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	// attach a permanent ban (PunishEnd == nil)
	var user model.User
	if err := testDB.Where("id = ?", database.TestUserCPSK1.ID).First(&user).Error; err != nil {
		t.Fatalf("failed to load user: %v", err)
	}
	now := time.Now()
	punishment := model.PunishmentStruct{PunishmentType: model.BanPunishment, PunishAt: &now, PunishEnd: nil}
	user.Punishment = &punishment
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("failed to save punishment: %v", err)
	}

	// Protected endpoint that also checks for ban punishment
	engine := gin.New()
	engine.GET("/protected-ban", RequireAuth(testDB), CheckPunishment(testDB, model.BanPunishment), checkUserHandler)

	req, _ := http.NewRequest(http.MethodGet, "/protected-ban", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	assert.Contains(t, body["error"], "permanent punishment")

	// cleanup punishment
	user.Punishment = nil
	user.PunishmentID = nil
	if err := testDB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("failed to cleanup punishment: %v", err)
	}
}

func TestCheckPunishment_NoUserInContext(t *testing.T) {
	engine := gin.New()
	engine.GET("/check", CheckPunishment(testDB, "ban"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "user information not provided")
}

func TestCheckPunishment_AdminUser(t *testing.T) {
	engine := gin.New()
	engine.GET("/check", RequireAuth(testDB), CheckPunishment(testDB, "ban"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Admin can access"})
	})

	token, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, "Admin can access", body["message"])
}

func TestCheckPunishment_UserWithNoPunishment(t *testing.T) {
	engine := gin.New()
	engine.GET("/check", RequireAuth(testDB), CheckPunishment(testDB, "ban"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "No punishment"})
	})

	token, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, "No punishment", body["message"])
}

func TestCheckPunishment_UserWithDifferentPunishmentType(t *testing.T) {
	// Create a user with suspend punishment
	punishEnd := time.Now().Add(24 * time.Hour)
	punishment := model.PunishmentStruct{
		PunishmentType: "suspend",
		PunishEnd:      &punishEnd,
	}
	if err := testDB.Create(&punishment).Error; err != nil {
		t.Fatal(err)
	}

	punishmentID := int(punishment.ID)
	testUser := model.User{
		Username:     "test_suspended_user",
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
			FirstName: "Test",
			LastName:  "Suspended",
		},
	}
	if err := testDB.Create(&cpskUser).Error; err != nil {
		t.Fatal(err)
	}

	// Checking for "ban" when user has "suspend" should allow access
	engine := gin.New()
	engine.GET("/check", RequireAuth(testDB), CheckPunishment(testDB, "ban"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Different punishment type"})
	})

	token, _, err := auth.GenerateStandardToken(testUser.ID)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, "Different punishment type", body["message"])

	// Cleanup
	testDB.Unscoped().Delete(&cpskUser)
	testDB.Unscoped().Delete(&testUser)
	testDB.Unscoped().Delete(&punishment)
}

func TestCheckPunishment_UserWithMatchingActivePunishment(t *testing.T) {
	// Create a user with active ban
	punishment := model.PunishmentStruct{
		PunishmentType: "ban",
		PunishEnd:      nil, // Permanent ban
	}
	if err := testDB.Create(&punishment).Error; err != nil {
		t.Fatal(err)
	}

	punishmentID := int(punishment.ID)
	testUser := model.User{
		Username:     "test_banned_user",
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
			FirstName: "Test",
			LastName:  "Banned",
		},
	}
	if err := testDB.Create(&cpskUser).Error; err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	engine.GET("/check", RequireAuth(testDB), CheckPunishment(testDB, "ban"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Should not reach here"})
	})

	token, _, err := auth.GenerateStandardToken(testUser.ID)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Contains(t, body["error"], "You don't have access to this endpoint due to permanent punishment")

	// Cleanup
	testDB.Unscoped().Delete(&cpskUser)
	testDB.Unscoped().Delete(&testUser)
	testDB.Unscoped().Delete(&punishment)
}

func TestCheckPunishment_UserWithExpiredPunishment(t *testing.T) {
	// Create a user with expired ban
	punishEnd := time.Now().Add(-24 * time.Hour) // Expired yesterday
	punishment := model.PunishmentStruct{
		PunishmentType: "ban",
		PunishEnd:      &punishEnd,
	}
	if err := testDB.Create(&punishment).Error; err != nil {
		t.Fatal(err)
	}

	punishmentID := int(punishment.ID)
	testUser := model.User{
		Username:     "test_expired_ban_user",
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
			FirstName: "Test",
			LastName:  "Expired",
		},
	}
	if err := testDB.Create(&cpskUser).Error; err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	engine.GET("/check", RequireAuth(testDB), CheckPunishment(testDB, "ban"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Punishment expired and removed"})
	})

	token, _, err := auth.GenerateStandardToken(testUser.ID)
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	// Should pass through since RemovePunishment removes expired punishments
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]interface{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, "Punishment expired and removed", body["message"])

	// Cleanup
	testDB.Unscoped().Delete(&cpskUser)
	testDB.Unscoped().Delete(&testUser)
	testDB.Unscoped().Delete(&punishment)
}

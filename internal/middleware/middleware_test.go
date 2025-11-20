package middleware

import (
	"HireMeMaybe-backend/internal/auth"
	"HireMeMaybe-backend/internal/database"
	"HireMeMaybe-backend/internal/model"
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


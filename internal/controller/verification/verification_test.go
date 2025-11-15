package verification

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

func TestAIVerifyCompany_Unauthorized(t *testing.T) {
	// Test without authentication
	r := gin.Default()
	jc := &VerificationController{
		DB: testDB,
	}
	r.POST("/company/ai-verify", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), jc.AIVerifyCompany)

	rec, resp := testutil.MakeJSONRequest(nil, "", r, "/company/ai-verify", http.MethodPost)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, resp["error"], "authorization header")
}

func TestAIVerifyCompany_WrongRole(t *testing.T) {
	// Test with CPSK user token (wrong role)
	cpskToken, err := auth.GetAccessToken(t, testDB, database.TestUserCPSK1.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	jc := &VerificationController{
		DB: testDB,
	}
	r.POST("/company/ai-verify", middleware.RequireAuth(testDB), middleware.CheckRole(model.RoleCompany), jc.AIVerifyCompany)

	rec, resp := testutil.MakeJSONRequest(nil, cpskToken, r, "/company/ai-verify", http.MethodPost)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, resp["error"], "permission")
}

func TestVerifyCompanyWithAI_ValidCompany(t *testing.T) {
	// Skip if no OpenAI API key is configured
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping AI verification test: OPENAI_API_KEY not configured")
	}

	// Create a test company with professional information
	testCompany := model.CompanyUser{
		EditableCompanyInfo: model.EditableCompanyInfo{
			Name:     "TechStart Solutions Inc",
			Overview: "We are a leading software development company specializing in enterprise solutions, cloud computing, and AI integration. Our team of experienced developers works with Fortune 500 companies.",
			Industry: "Technology",
			Size:     testutil.StringPtr("M"),
		},
		User: model.User{
			Email: testutil.StringPtr("contact@techstart.com"),
			EditableUserInfo: model.EditableUserInfo{
				Tel: testutil.StringPtr("+1-555-0123"),
			},
		},
	}

	result, err := VerifyCompanyWithAI(testCompany)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Reasoning)
	assert.Contains(t, []string{"High", "Medium", "Low"}, result.Confidence)
	// Should likely verify this professional company
	assert.True(t, result.ShouldVerify)
}

func TestVerifyCompanyWithAI_TestCompany(t *testing.T) {
	// Skip if no OpenAI API key is configured
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping AI verification test: OPENAI_API_KEY not configured")
	}

	// Create a test company with obvious test data
	testCompany := model.CompanyUser{
		EditableCompanyInfo: model.EditableCompanyInfo{
			Name:     "Test Company",
			Overview: "This is a test company for testing purposes.",
			Industry: "Testing",
			Size:     testutil.StringPtr("XS"),
		},
		User: model.User{
			Email: testutil.StringPtr("test@test.com"),
			EditableUserInfo: model.EditableUserInfo{
				Tel: testutil.StringPtr("123456"),
			},
		},
	}

	result, err := VerifyCompanyWithAI(testCompany)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Reasoning)
	assert.Contains(t, []string{"High", "Medium", "Low"}, result.Confidence)
	// Should NOT verify obvious test company
	assert.False(t, result.ShouldVerify)
}

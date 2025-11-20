package punishment

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

func TestPunishUser_UserNotFound(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	pc := &PunishmentController{DB: testDB}
	r.PUT("/punish/:user_id", middleware.RequireAuth(testDB), middleware.CheckRole("admin"), pc.PunishUser)

	body := gin.H{"type": "ban"}

	rec, resp := testutil.MakeJSONRequest(body, adminToken, r, "/punish/00000000-0000-0000-0000-000000000000", http.MethodPut)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "User not found")
	}
}

func TestPunishUser_AdminForbidden(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	pc := &PunishmentController{DB: testDB}
	r.PUT("/punish/:user_id", middleware.RequireAuth(testDB), middleware.CheckRole("admin"), pc.PunishUser)

	// try to punish the seeded admin user
	body := gin.H{"type": "ban"}

	rec, resp := testutil.MakeJSONRequest(body, adminToken, r, "/punish/"+database.TestAdminUser.ID.String(), http.MethodPut)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "Unable to punish other admin")
	}
}

func TestDeletePunishmentRecord_UserNotFound(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	pc := &PunishmentController{DB: testDB}
	r.DELETE("/punish/:user_id", middleware.RequireAuth(testDB), middleware.CheckRole("admin"), pc.DeletePunishmentRecord)

	rec, resp := testutil.MakeJSONRequest(nil, adminToken, r, "/punish/00000000-0000-0000-0000-000000000000", http.MethodDelete)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	if resp != nil {
		assert.Contains(t, resp["error"], "User not found")
	}
}

func TestDeletePunishmentRecord_Success(t *testing.T) {
	adminToken, err := auth.GetAccessToken(t, testDB, database.TestAdminUser.Username, database.TestSeedPassword)
	assert.NoError(t, err)

	r := gin.Default()
	pc := &PunishmentController{DB: testDB}
	r.DELETE("/punish/:user_id", middleware.RequireAuth(testDB), middleware.CheckRole("admin"), pc.DeletePunishmentRecord)

	// delete punishment for an existing seeded user (may be no-op but should return 200)
	rec, _ := testutil.MakeJSONRequest(nil, adminToken, r, "/punish/"+database.TestUserCPSK1.ID.String(), http.MethodDelete)

	assert.Equal(t, http.StatusOK, rec.Code)
}

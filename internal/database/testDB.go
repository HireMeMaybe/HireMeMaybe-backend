package database

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	// Load env
	_ "github.com/joho/godotenv/autoload"

	m "HireMeMaybe-backend/internal/model"
	"HireMeMaybe-backend/internal/utilities"
)

var testDBInstance *DBinstanceStruct
var teardown func(context.Context, ...testcontainers.TerminateOption) error

// Exported test users & profiles
var (
	TestAdminUser    m.User
	TestUserCPSK1    m.User
	TestUserCPSK2    m.User
	TestUserCompany1 m.User
	TestUserCompany2 m.User
	TestCPSK1        m.CPSKUser
	TestCPSK2        m.CPSKUser
	TestCompany1     m.Company
	TestCompany2     m.Company

	// Add exported plain password
	TestSeedPassword = "SeedPass123!"
)

// GetTestDB starts a PostgreSQL test container and returns a teardown function,
// the DB instance, and any error encountered during setup.
func GetTestDB() (func(context.Context, ...testcontainers.TerminateOption) error, *DBinstanceStruct, error) {

	if testDBInstance != nil && teardown != nil {
		return teardown, testDBInstance, nil
	}

	// Database configuration
	var (
		dbName = "database"
		dbPwd  = "password"
		dbUser = "user"
	)

	dbContainer, err := postgres.Run(
		context.Background(),
		"postgres:latest",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, nil, err
	}

	dbHost, err := dbContainer.Host(context.Background())
	if err != nil {
		return dbContainer.Terminate, nil, err
	}

	dbPort, err := dbContainer.MappedPort(context.Background(), nat.Port("5432/tcp"))
	if err != nil {
		return dbContainer.Terminate, nil, err
	}

	config := &DBConfig{
		useConstr: true,
		Constr:    fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort.Port(), dbUser, dbPwd, dbName),
	}

	db, err := NewDBInstance(config)
	if err != nil {
		return dbContainer.Terminate, nil, err
	}

	// Seed sample CPSK students and company users
	if err := seedTestData(db); err != nil {
		_ = dbContainer.Terminate(context.Background())
		return nil, nil, err
	}

	testDBInstance = db
	teardown = dbContainer.Terminate

	return dbContainer.Terminate, db, nil
}

// seedTestData inserts sample CPSKUser and Company records (2 each) if empty.
func seedTestData(db *DBinstanceStruct) error {
	var userCount int64
	if err := db.Model(&m.User{}).Count(&userCount).Error; err != nil {
		return err
	}

	// Ignore admin user that got create during NewDBInstance
	if userCount > 1 {
		return loadTestData(db)
	}

	// Base data
	tels := []*string{ptr("0100000001"), ptr("0100000002"), ptr("0200000001"), ptr("0200000002"), ptr("0300000001")}
	emails := []*string{ptr("student1@example.com"), ptr("student2@example.com"), ptr("company1@example.com"), ptr("company2@example.com"), ptr("admin@example.com")}
	userSpecs := []struct {
		username string
		email    *string
		tel      *string
		role     string
	}{
		{"cpsk_student_1", emails[0], tels[0], m.RoleCPSK},
		{"cpsk_student_2", emails[1], tels[1], m.RoleCPSK},
		{"company_user_1", emails[2], tels[2], m.RoleCompany},
		{"company_user_2", emails[3], tels[3], m.RoleCompany},
		{"admin_user", emails[4], tels[4], m.RoleAdmin},
	}

	// Pre-hash shared password for all seeded users
	hashedPwd, errHash := utilities.HashPassword(TestSeedPassword)
	if errHash != nil {
		return errHash
	}

	users := make([]m.User, 0, len(userSpecs))
	for _, s := range userSpecs {
		users = append(users, m.User{
			ID:             uuid.New(),
			Username:       s.username,
			Email:          s.email,
			Tel:            s.tel,
			Role:           s.role,
			Password:       hashedPwd,
			ProfilePicture: "",
		})
	}

	if err := db.Create(&users).Error; err != nil {
		return err
	}

	// Map created users to exported variables
	for _, u := range users {
		switch u.Username {
		case "cpsk_student_1":
			TestUserCPSK1 = u
		case "cpsk_student_2":
			TestUserCPSK2 = u
		case "company_user_1":
			TestUserCompany1 = u
		case "company_user_2":
			TestUserCompany2 = u
		case "admin_user":
			TestAdminUser = u
		}
	}

	progCPE, progSKE := "CPE", "SKE"
	year3, year2 := "3", "2"
	sizeM, sizeL := "M", "L"

	cpskProfiles := []m.CPSKUser{
		{
			UserID:           TestUserCPSK1.ID,
			FirstName:        "Alice",
			LastName:         "Nguyen",
			Program:          &progCPE,
			EducationalLevel: &year3,
			SoftSkill:        pq.StringArray{"Teamwork", "Communication"},
		},
		{
			UserID:           TestUserCPSK2.ID,
			FirstName:        "Bob",
			LastName:         "Somsak",
			Program:          &progSKE,
			EducationalLevel: &year2,
			SoftSkill:        pq.StringArray{"Problem Solving", "Adaptability"},
		},
	}
	if err := db.Create(&cpskProfiles).Error; err != nil {
		return err
	}

	companies := []m.Company{
		{
			UserID:         TestUserCompany1.ID,
			VerifiedStatus: m.StatusVerified,
			Name:           "TechNova",
			Overview:       "Innovative platform solutions",
			Industry:       "Software",
			Size:           &sizeM,
		},
		{
			UserID:         TestUserCompany2.ID,
			VerifiedStatus: m.StatusPending,
			Name:           "DataForge",
			Overview:       "Data analytics consulting",
			Industry:       "Consulting",
			Size:           &sizeL,
		},
	}
	if err := db.Create(&companies).Error; err != nil {
		return err
	}

	// Assign exported profile structs
	TestCPSK1 = cpskProfiles[0]
	TestCPSK2 = cpskProfiles[1]
	TestCompany1 = companies[0]
	TestCompany2 = companies[1]

	return nil
}

// loadTestData populates exported variables when records already exist.
func loadTestData(db *DBinstanceStruct) error {
	var users []m.User
	if err := db.Where("username IN ?", []string{
		"cpsk_student_1", "cpsk_student_2", "company_user_1", "company_user_2",
	}).Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		switch u.Username {
		case "cpsk_student_1":
			TestUserCPSK1 = u
		case "cpsk_student_2":
			TestUserCPSK2 = u
		case "company_user_1":
			TestUserCompany1 = u
		case "company_user_2":
			TestUserCompany2 = u
		}
	}

	// Load CPSK profiles
	if err := db.Where("user_id IN ?", []uuid.UUID{TestUserCPSK1.ID, TestUserCPSK2.ID}).Find(&[]*m.CPSKUser{&TestCPSK1, &TestCPSK2}).Error; err != nil {
		// Fallback individual queries
		_ = db.First(&TestCPSK1, "user_id = ?", TestUserCPSK1.ID).Error
		_ = db.First(&TestCPSK2, "user_id = ?", TestUserCPSK2.ID).Error
	}

	// Load Company profiles
	if err := db.Where("user_id IN ?", []uuid.UUID{TestUserCompany1.ID, TestUserCompany2.ID}).Find(&[]*m.Company{&TestCompany1, &TestCompany2}).Error; err != nil {
		_ = db.First(&TestCompany1, "user_id = ?", TestUserCompany1.ID).Error
		_ = db.First(&TestCompany2, "user_id = ?", TestUserCompany2.ID).Error
	}

	return nil
}

// ptr helper
func ptr[T any](v T) *T { return &v }

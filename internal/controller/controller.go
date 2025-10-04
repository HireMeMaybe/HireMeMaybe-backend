package controller

import "HireMeMaybe-backend/internal/database"

// JobController struct holds the database connection for job-related operations.
type JobController struct {
	DB *database.DBinstanceStruct
}

// NewJobController creates a new instance of JobController with the provided database connection.
func NewJobController(db *database.DBinstanceStruct) *JobController {
	return &JobController{
		DB: db,
	}
}

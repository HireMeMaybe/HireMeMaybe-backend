package controller

import "HireMeMaybe-backend/internal/database"

type JobController struct {
	DB *database.DBinstanceStruct
}

func NewJobController(db *database.DBinstanceStruct) *JobController {
	return &JobController{
		DB: db,
	}
}

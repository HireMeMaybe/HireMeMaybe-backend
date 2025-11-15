package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// EditableJobPostInfo is part of job post that can be edited
type EditableJobPostInfo struct {
	Title    string         `gorm:"type:text" json:"title"`
	Desc     string         `gorm:"type:text" json:"desc"`
	Req      string         `gorm:"type:text" json:"req"`
	ExpLvl   string         `gorm:"type:text" json:"exp_lvl"`
	Location string         `gorm:"type:text" json:"location"`
	Type     string         `gorm:"type:text" json:"type"`
	Salary   string         `gorm:"type:text" json:"salary"`
	Tags     pq.StringArray `gorm:"type:text[]" json:"tags"`
	Expiring *time.Time     `gorm:"type:timestamp" json:"expiring,omitempty"`
}

// JobPost is gorm model for store job post data in DB
type JobPost struct {
	ID            uint        `gorm:"primaryKey;autoIncrement;->" json:"id"`
	CompanyUserID uuid.UUID   `gorm:"not null;index;<-:create" json:"company_id"`
	CompanyUser   CompanyUser `gorm:"foreignKey:CompanyUserID;references:UserID" json:"company_user"`
	EditableJobPostInfo
	PostTime      time.Time      `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;->" json:"post_time"`
	Applications  []Application  `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE" json:"applications"`
	DefaultForm   bool           `gorm:"type:boolean;default:true" json:"default_form"`
	OptionalForms pq.StringArray `gorm:"type:text[]" json:"optional_forms"`
}

// JobPostResponse is the response struct for job post with user application status
type JobPostResponse struct {
	ID            uint        `json:"id"`
	CompanyUserID uuid.UUID   `json:"company_id"`
	CompanyUser   CompanyUser `json:"company_user"`
	PostTime      time.Time   `json:"post_time"`
	UserApply     bool        `json:"user_apply"`
	EditableJobPostInfo
}

// ToJobPostResponse converts JobPost to JobPostResponse
func (j *JobPost) ToJobPostResponse(user User) (JobPostResponse, error) {

	var resp JobPostResponse

	b, err := json.Marshal(j)
	if err != nil {
		return resp, err
	}

	err = json.Unmarshal(b, &resp)
	if err != nil {
		return resp, err
	}

	userApply := false

	if user.Role == RoleCPSK {
		for _, application := range j.Applications {
			if application.CPSKID.String() == user.ID.String() {
				userApply = true
				break
			}
		}
	}
	resp.UserApply = userApply

	return resp, nil
}

package auth

import (
	"HireMeMaybe-backend/internal/model"
)

type cpskResponse struct {
	User        model.CPSKUser `json:"user"`
	AccessToken string         `json:"access_token"`
}

func (r *cpskResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}


type companyResponse struct {
	User        model.CompanyUser `json:"user"`
	AccessToken string        `json:"access_token"`
}

func (r *companyResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}


type visitorResponse struct {
	User        model.VisitorUser `json:"user"`
	AccessToken string        `json:"access_token"`
}

func (r *visitorResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

func (r *adminResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}


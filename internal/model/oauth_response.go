package model

type AccessToke struct {
	Token string `json:"token"`
}

type CPSKResponse struct {
	User        CPSKUser `json:"user"`
	AccessToken string   `json:"access_token"`
}

func (r *CPSKResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

type CompanyResponse struct {
	User        CompanyUser `json:"user"`
	AccessToken string  `json:"access_token"`
}

func (r *CompanyResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

type VisitorResponse struct {
	User        VisitorUser `json:"user"`
	AccessToken string  `json:"access_token"`
}

func (r *VisitorResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

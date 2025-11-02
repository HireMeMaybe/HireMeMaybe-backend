package model

// AccessToken struct holds the access token data
type AccessToken struct {
	Token string `json:"token"`
}

// CPSKResponse struct holds the response data for CPSK student user login or registration
type CPSKResponse struct {
	User        CPSKUser `json:"user"`
	AccessToken string   `json:"access_token"`
}

// SetAccessToken sets the access token in the CPSKResponse
func (r *CPSKResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

// CompanyResponse struct holds the response data for Company user login or registration
type CompanyResponse struct {
	User        CompanyUser `json:"user"`
	AccessToken string      `json:"access_token"`
}

// SetAccessToken sets the access token in the CompanyResponse
func (r *CompanyResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

// VisitorResponse struct holds the response data for Visitor user login or registration
type VisitorResponse struct {
	User        VisitorUser `json:"user"`
	AccessToken string      `json:"access_token"`
}

// SetAccessToken sets the access token in the VisitorResponse
func (r *VisitorResponse) SetAccessToken(accessToken string) {
	r.AccessToken = accessToken
}

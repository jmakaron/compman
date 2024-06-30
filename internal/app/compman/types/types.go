package types

import "encoding/json"

type User struct {
	ID       string
	Username string
	Email    string
	Password string
}

type CompanyType int

const (
	CompanyTypeCorporation CompanyType = iota
	CompanyTypeNonProfit
	CompanyTypeCooperative
	CompanyTypeSoleProprietorship
)

func (c CompanyType) String() string {
	var r string
	switch c {
	case CompanyTypeCorporation:
		r = "corporation"
	case CompanyTypeNonProfit:
		r = "non-profit"
	case CompanyTypeCooperative:
		r = "cooperative"
	case CompanyTypeSoleProprietorship:
		r = "sole-proprietorship"
	default:
		r = ""
	}
	return r
}

func ParseCompanyType(str string) CompanyType {
	var r CompanyType
	switch str {
	case "corporation":
		r = CompanyTypeCorporation
	case "non-profit":
		r = CompanyTypeNonProfit
	case "cooperative":
		r = CompanyTypeCooperative
	case "sole-proprietorship":
		r = CompanyTypeSoleProprietorship
	default:
		r = -1
	}
	return r
}
func (c CompanyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

func (c *CompanyType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = ParseCompanyType(s)
	return nil
}

type Company struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Desc        *string     `json:"description"`
	EmployeeCnt int         `json:"employee_count"`
	Registered  bool        `json:"registered"`
	CType       CompanyType `json:"type"`
}

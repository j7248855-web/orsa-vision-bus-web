package models

type CreateUsers struct {
	Login       string `json:"login" db:"username"`
	Password    string `json:"password" db:"password"`
	FullName    string `json:"fullname" db:"full_name"`
	Permissions bool   `json:"permissions"`
}

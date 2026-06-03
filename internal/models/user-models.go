package models

type CreateUsers struct {
	ID          int    `json:"id" db:"id"`
	Login       string `json:"login" db:"username" binding:"required"`
	Password    string `json:"password" db:"password"`
	FullName    string `json:"fullname" db:"full_name" binding:"required"`
	Permissions bool   `json:"permissions" db:"permissions"`
}

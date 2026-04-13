package models

type Login struct {
	ID       int    `json:"id" db:"id"`
	Name     string `json:"name" db:"user"`
	Password string `json:"password" db:"password"`
}

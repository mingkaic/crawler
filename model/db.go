// file: db.go

// Package model ...
// Is a db api for xcrawl
package model

import (
	"github.com/jinzhu/gorm"
)

type Site struct {
	gorm.Model
	Link string
}

var Db *gorm.DB

// InitDB ...
// Initializes global db
func InitDB(dialect, source string) {
	Db, err := gorm.Open(dialect, source)
	if err != nil {
		panic(err)
	}
	Db.AutoMigrate(&Site{})
}

func AddLink(URI string) {
	Db.Create(&Site{Link: URI})
}

func RmLink(URI string) {
	var site Site
	Db.First(&site, "link = ?", URI)
	Db.Delete(&site)
}

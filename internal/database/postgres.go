package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
)

type Post struct {
	ID        string `gorm:"primaryKey"`
	AuthorID  string
	Author    Author `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
	Body      string
	CreatedAt string
}
type Author struct {
	ID       string `gorm:"primaryKey"`
	Nickname string
	Avatar   string
}

var DB *gorm.DB

func InitPostgres() {
	dsn := "host=localhost user=postgres password=postgres dbname=blog port=5432 sslmode=disable TimeZone=UTC"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to Postgres: %v", err)
	}

	err = DB.AutoMigrate(&Author{}, &Post{})
	if err != nil {
		log.Fatalf("failed to run AutoMigrate: %v", err)
	}
}

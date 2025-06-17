package storage

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"os"
)

type Storage struct {
	db *gorm.DB
}

func InitDB() (*Storage, error) {
	dsn := os.Getenv("MYSQL_DSN")

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// TODO implement migrations here if needed

	return &Storage{db: db}, nil

}

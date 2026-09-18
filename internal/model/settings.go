package models

import "gorm.io/gorm"

type AppSetting struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex"`
	Value string
}

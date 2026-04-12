package models

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Slug      string         `gorm:"size:64;uniqueIndex;not null" json:"slug"`
	SortOrder int            `gorm:"not null;default:0" json:"sortOrder"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Expense struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UserID      *uint          `gorm:"index" json:"-"` // ileride auth; şimdilik NULL
	CategoryID  uint           `gorm:"index;not null" json:"categoryId"`
	Category    Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	AmountMinor int64          `gorm:"not null" json:"-"` // kuruş (TRY)
	Note        string         `gorm:"size:512" json:"note"`
	OccurredAt  time.Time      `gorm:"index;not null" json:"occurredAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

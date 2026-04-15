package models

import (
	"time"

	"gorm.io/gorm"
)

// ExpenseList is a user-owned expense sheet (multi-list MVP).
type ExpenseList struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	ClerkUserID *string        `gorm:"size:191;index;not null" json:"-"` // owner Clerk subject
	Name        string         `gorm:"size:128;not null" json:"name"`
	IsFavorite  bool           `gorm:"not null;default:false" json:"isFavorite"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

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
	ID           uint           `gorm:"primaryKey" json:"id"`
	ListID       *uint          `gorm:"index" json:"listId,omitempty"`
	UserID       *uint          `gorm:"index" json:"-"` // reserved for internal user PK
	ClerkUserID  *string        `gorm:"size:191;index" json:"-"` // Clerk sub; set when using BFF
	CategoryID   uint           `gorm:"index;not null" json:"categoryId"`
	Category    Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	AmountMinor int64          `gorm:"not null" json:"-"` // TRY minor units (e.g. cents)
	Note        string         `gorm:"size:512" json:"note"`
	OccurredAt  time.Time      `gorm:"index;not null" json:"occurredAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

package db

import (
	"errors"
	"fmt"

	"expenses-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	return gdb, nil
}

func Migrate(gdb *gorm.DB) error {
	return gdb.AutoMigrate(&models.Category{}, &models.ExpenseList{}, &models.Expense{})
}

var defaultCategories = []models.Category{
	{Name: "Groceries", Slug: "market", SortOrder: 10},
	{Name: "Transport", Slug: "ulasim", SortOrder: 20},
	{Name: "Bills", Slug: "faturalar", SortOrder: 30},
	{Name: "Food & drink", Slug: "yeme-icme", SortOrder: 40},
	{Name: "Entertainment", Slug: "eglence", SortOrder: 50},
	{Name: "Health", Slug: "saglik", SortOrder: 60},
	{Name: "Other", Slug: "diger", SortOrder: 999},
}

func SeedCategories(gdb *gorm.DB) error {
	var n int64
	if err := gdb.Model(&models.Category{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for _, c := range defaultCategories {
		if err := gdb.Create(&c).Error; err != nil {
			return fmt.Errorf("seed category %s: %w", c.Slug, err)
		}
	}
	return nil
}

func GetCategoryByID(gdb *gorm.DB, id uint) (*models.Category, error) {
	var c models.Category
	if err := gdb.First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

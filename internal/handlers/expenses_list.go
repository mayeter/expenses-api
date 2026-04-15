package handlers

import (
	"errors"
	"math"
	"time"

	"expenses-backend/internal/db"
	"expenses-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ListExpensesInList GET /api/lists/:listId/expenses
func (h *Handlers) ListExpensesInList(c *fiber.Ctx) error {
	listID, err := parseListID(c)
	if err != nil {
		return err
	}
	if _, err := h.ownedList(c, listID); err != nil {
		return err
	}
	fromStr := c.Query("from")
	toStr := c.Query("to")

	q := h.expenseQueryForList(c, listID).Preload("Category").Order("occurred_at DESC, id DESC")

	if fromStr != "" {
		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid from (use YYYY-MM-DD)")
		}
		q = q.Where("occurred_at >= ?", from.UTC())
	}
	if toStr != "" {
		to, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid to (use YYYY-MM-DD)")
		}
		end := to.Add(24 * time.Hour)
		q = q.Where("occurred_at < ?", end.UTC())
	}

	var rows []models.Expense
	if err := q.Find(&rows).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	out := make([]expenseResponse, 0, len(rows))
	for _, e := range rows {
		out = append(out, toExpenseResponse(e))
	}
	return c.JSON(out)
}

// CreateExpenseInList POST /api/lists/:listId/expenses
func (h *Handlers) CreateExpenseInList(c *fiber.Ctx) error {
	listID, err := parseListID(c)
	if err != nil {
		return err
	}
	if _, err := h.ownedList(c, listID); err != nil {
		return err
	}

	var body createExpenseBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON")
	}
	if body.CategoryID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "categoryId required")
	}
	cat, err := db.GetCategoryByID(h.DB, body.CategoryID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if cat == nil {
		return fiber.NewError(fiber.StatusBadRequest, "unknown categoryId")
	}

	minor := int64(math.Round(body.Amount * 100))
	if minor <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "amount must be positive")
	}

	var occurred time.Time
	if body.OccurredAt != "" {
		occurred, err = time.Parse(time.RFC3339, body.OccurredAt)
		if err != nil {
			occurred, err = time.Parse("2006-01-02", body.OccurredAt)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "occurredAt must be RFC3339 or YYYY-MM-DD")
			}
		}
	} else {
		occurred = time.Now().UTC()
	}

	lid := listID
	exp := models.Expense{
		ListID:      &lid,
		CategoryID:  body.CategoryID,
		AmountMinor: minor,
		Note:        body.Note,
		OccurredAt:  occurred.UTC(),
	}
	if uid, ok := clerkUserID(c); ok {
		exp.ClerkUserID = &uid
	}
	if err := h.DB.Create(&exp).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := h.expenseQueryForList(c, listID).Preload("Category").First(&exp, exp.ID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(toExpenseResponse(exp))
}

// UpdateExpenseInList PATCH /api/lists/:listId/expenses/:id
func (h *Handlers) UpdateExpenseInList(c *fiber.Ctx) error {
	listID, err := parseListID(c)
	if err != nil {
		return err
	}
	if _, err := h.ownedList(c, listID); err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	var body updateExpenseBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON")
	}
	if body.Amount == nil && body.CategoryID == nil && body.Note == nil && body.OccurredAt == nil {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}

	q := h.expenseQueryForList(c, listID).Preload("Category")
	var exp models.Expense
	if err := q.Where("id = ?", uint(id)).First(&exp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	updates := map[string]interface{}{}
	if body.Amount != nil {
		minor := int64(math.Round(*body.Amount * 100))
		if minor <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "amount must be positive")
		}
		updates["amount_minor"] = minor
	}
	if body.CategoryID != nil {
		if *body.CategoryID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "categoryId invalid")
		}
		cat, err := db.GetCategoryByID(h.DB, *body.CategoryID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if cat == nil {
			return fiber.NewError(fiber.StatusBadRequest, "unknown categoryId")
		}
		updates["category_id"] = *body.CategoryID
	}
	if body.Note != nil {
		updates["note"] = *body.Note
	}
	if body.OccurredAt != nil && *body.OccurredAt != "" {
		var occurred time.Time
		occurred, err = time.Parse(time.RFC3339, *body.OccurredAt)
		if err != nil {
			occurred, err = time.Parse("2006-01-02", *body.OccurredAt)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "occurredAt must be RFC3339 or YYYY-MM-DD")
			}
		}
		updates["occurred_at"] = occurred.UTC()
	}

	if len(updates) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}
	if err := h.DB.Model(&exp).Where("id = ?", uint(id)).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := h.expenseQueryForList(c, listID).Preload("Category").First(&exp, uint(id)).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(toExpenseResponse(exp))
}

// DeleteExpenseInList DELETE /api/lists/:listId/expenses/:id
func (h *Handlers) DeleteExpenseInList(c *fiber.Ctx) error {
	listID, err := parseListID(c)
	if err != nil {
		return err
	}
	if _, err := h.ownedList(c, listID); err != nil {
		return err
	}
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	res := h.expenseQueryForList(c, listID).Where("id = ?", uint(id)).Delete(&models.Expense{})
	if res.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

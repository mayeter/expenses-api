package handlers

import (
	"errors"
	"strings"

	"expenses-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const defaultExpenseListName = "Personal"

func (h *Handlers) ownedList(c *fiber.Ctx, listID uint) (*models.ExpenseList, error) {
	var list models.ExpenseList
	if err := h.DB.First(&list, listID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "list not found")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	uid, ok := clerkUserID(c)
	if ok {
		if list.ClerkUserID == nil || *list.ClerkUserID != uid {
			return nil, fiber.NewError(fiber.StatusForbidden, "forbidden")
		}
	}
	return &list, nil
}

func (h *Handlers) ensureDefaultList(uid string) (*models.ExpenseList, error) {
	var n int64
	if err := h.DB.Model(&models.ExpenseList{}).Where("clerk_user_id = ?", uid).Count(&n).Error; err != nil {
		return nil, err
	}
	if n > 0 {
		return nil, nil
	}
	u := uid
	row := models.ExpenseList{Name: defaultExpenseListName, IsFavorite: true, ClerkUserID: &u}
	if err := h.DB.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListExpenseLists GET /api/lists?scope=mine|shared (shared is empty until sharing ships).
func (h *Handlers) ListExpenseLists(c *fiber.Ctx) error {
	uid, ok := clerkUserID(c)
	if !ok {
		var all []models.ExpenseList
		if err := h.DB.Order("is_favorite DESC, name ASC, id ASC").Find(&all).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(all)
	}

	if _, err := h.ensureDefaultList(uid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	scope := strings.TrimSpace(strings.ToLower(c.Query("scope", "mine")))
	if scope == "shared" {
		return c.JSON([]models.ExpenseList{})
	}

	var lists []models.ExpenseList
	if err := h.DB.Where("clerk_user_id = ?", uid).Order("is_favorite DESC, name ASC, id ASC").Find(&lists).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(lists)
}

type createExpenseListBody struct {
	Name       string `json:"name"`
	IsFavorite bool   `json:"isFavorite"`
}

// CreateExpenseList POST /api/lists
func (h *Handlers) CreateExpenseList(c *fiber.Ctx) error {
	uid, ok := clerkUserID(c)
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "clerk context required")
	}
	var body createExpenseListBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON")
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	u := uid
	row := models.ExpenseList{Name: name, IsFavorite: body.IsFavorite, ClerkUserID: &u}
	if err := h.DB.Create(&row).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(row)
}

type patchExpenseListBody struct {
	Name       *string `json:"name"`
	IsFavorite *bool   `json:"isFavorite"`
}

// UpdateExpenseList PATCH /api/lists/:listId
func (h *Handlers) UpdateExpenseList(c *fiber.Ctx) error {
	listID, err := c.ParamsInt("listId")
	if err != nil || listID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid listId")
	}
	if _, err := h.ownedList(c, uint(listID)); err != nil {
		return err
	}
	var body patchExpenseListBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON")
	}
	if body.Name == nil && body.IsFavorite == nil {
		return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
	}
	updates := map[string]interface{}{}
	if body.Name != nil {
		n := strings.TrimSpace(*body.Name)
		if n == "" {
			return fiber.NewError(fiber.StatusBadRequest, "name invalid")
		}
		updates["name"] = n
	}
	if body.IsFavorite != nil {
		updates["is_favorite"] = *body.IsFavorite
	}
	if err := h.DB.Model(&models.ExpenseList{}).Where("id = ?", uint(listID)).Updates(updates).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var out models.ExpenseList
	if err := h.DB.First(&out, uint(listID)).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(out)
}

// DeleteExpenseList DELETE /api/lists/:listId (owner only; soft-deletes expenses in the list).
func (h *Handlers) DeleteExpenseList(c *fiber.Ctx) error {
	listID, err := c.ParamsInt("listId")
	if err != nil || listID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid listId")
	}
	if _, err := h.ownedList(c, uint(listID)); err != nil {
		return err
	}
	tx := h.DB.Begin()
	if err := tx.Model(&models.Expense{}).Where("list_id = ?", uint(listID)).Delete(&models.Expense{}).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Delete(&models.ExpenseList{}, uint(listID)).Error; err != nil {
		tx.Rollback()
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := tx.Commit().Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func parseListID(c *fiber.Ctx) (uint, error) {
	id, err := c.ParamsInt("listId")
	if err != nil || id <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid listId")
	}
	return uint(id), nil
}

func (h *Handlers) expenseQueryForList(c *fiber.Ctx, listID uint) *gorm.DB {
	q := h.DB.Model(&models.Expense{}).Where("list_id = ?", listID)
	if id, ok := clerkUserID(c); ok {
		q = q.Where("clerk_user_id = ?", id)
	}
	return q
}

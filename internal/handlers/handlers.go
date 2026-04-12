package handlers

import (
	"errors"
	"math"
	"strings"
	"time"

	"expenses-backend/internal/db"
	"expenses-backend/internal/middleware"
	"expenses-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"gorm.io/gorm"
)

type Handlers struct {
	DB *gorm.DB
}

func Register(app *fiber.App, gdb *gorm.DB, allowedOrigins, internalSecret string) {
	h := &Handlers{DB: gdb}

	parts := strings.Split(allowedOrigins, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	origins := strings.Join(parts, ",")
	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, X-Internal-Secret, X-Clerk-User-Id",
		AllowCredentials: origins != "*" && origins != "",
	}))

	app.Get("/health", h.Health)
	api := app.Group("/api")
	api.Use(middleware.InternalAuth(internalSecret))
	api.Get("/categories", h.ListCategories)
	api.Get("/expenses", h.ListExpenses)
	api.Post("/expenses", h.CreateExpense)
	api.Patch("/expenses/:id", h.UpdateExpense)
	api.Delete("/expenses/:id", h.DeleteExpense)
}

func (h *Handlers) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handlers) ListCategories(c *fiber.Ctx) error {
	var list []models.Category
	if err := h.DB.Order("sort_order ASC, id ASC").Find(&list).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

type expenseResponse struct {
	ID         uint           `json:"id"`
	UserID     *uint          `json:"userId,omitempty"`
	CategoryID uint           `json:"categoryId"`
	Category   categoryBrief  `json:"category,omitempty"`
	Amount     float64        `json:"amount"`
	Currency   string         `json:"currency"`
	Note       string         `json:"note"`
	OccurredAt time.Time      `json:"occurredAt"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type categoryBrief struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func toExpenseResponse(e models.Expense) expenseResponse {
	out := expenseResponse{
		ID:         e.ID,
		UserID:     e.UserID,
		CategoryID: e.CategoryID,
		Amount:     float64(e.AmountMinor) / 100,
		Currency:   "TRY",
		Note:       e.Note,
		OccurredAt: e.OccurredAt,
		CreatedAt:  e.CreatedAt,
	}
	if e.Category.ID != 0 {
		out.Category = categoryBrief{ID: e.Category.ID, Name: e.Category.Name, Slug: e.Category.Slug}
	}
	return out
}

func clerkUserID(c *fiber.Ctx) (string, bool) {
	v := c.Locals(middleware.CtxClerkUserID)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func (h *Handlers) expenseBaseQuery(c *fiber.Ctx) *gorm.DB {
	q := h.DB.Model(&models.Expense{})
	if id, ok := clerkUserID(c); ok {
		q = q.Where("clerk_user_id = ?", id)
	}
	return q
}

func (h *Handlers) ListExpenses(c *fiber.Ctx) error {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	q := h.expenseBaseQuery(c).Preload("Category").Order("occurred_at DESC, id DESC")

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

type createExpenseBody struct {
	Amount      float64 `json:"amount"`
	CategoryID  uint    `json:"categoryId"`
	Note        string  `json:"note"`
	OccurredAt  string  `json:"occurredAt"`
}

func (h *Handlers) CreateExpense(c *fiber.Ctx) error {
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

	exp := models.Expense{
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
	if err := h.expenseBaseQuery(c).Preload("Category").First(&exp, exp.ID).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(toExpenseResponse(exp))
}

type updateExpenseBody struct {
	Amount     *float64 `json:"amount"`
	CategoryID *uint    `json:"categoryId"`
	Note       *string  `json:"note"`
	OccurredAt *string  `json:"occurredAt"`
}

func (h *Handlers) UpdateExpense(c *fiber.Ctx) error {
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

	q := h.expenseBaseQuery(c).Preload("Category")
	var exp models.Expense
	if err := q.First(&exp, uint(id)).Error; err != nil {
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
	if err := h.expenseBaseQuery(c).Preload("Category").First(&exp, uint(id)).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(toExpenseResponse(exp))
}

func (h *Handlers) DeleteExpense(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	res := h.expenseBaseQuery(c).Where("id = ?", uint(id)).Delete(&models.Expense{})
	if res.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, res.Error.Error())
	}
	if res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

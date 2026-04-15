package handlers

import (
	"strings"
	"time"

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

	api.Get("/lists/:listId/expenses", h.ListExpensesInList)
	api.Post("/lists/:listId/expenses", h.CreateExpenseInList)
	api.Patch("/lists/:listId/expenses/:id", h.UpdateExpenseInList)
	api.Delete("/lists/:listId/expenses/:id", h.DeleteExpenseInList)
	api.Patch("/lists/:listId", h.UpdateExpenseList)
	api.Delete("/lists/:listId", h.DeleteExpenseList)
	api.Get("/lists", h.ListExpenseLists)
	api.Post("/lists", h.CreateExpenseList)
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

type createExpenseBody struct {
	Amount      float64 `json:"amount"`
	CategoryID  uint    `json:"categoryId"`
	Note        string  `json:"note"`
	OccurredAt  string  `json:"occurredAt"`
}

type updateExpenseBody struct {
	Amount     *float64 `json:"amount"`
	CategoryID *uint    `json:"categoryId"`
	Note       *string  `json:"note"`
	OccurredAt *string  `json:"occurredAt"`
}

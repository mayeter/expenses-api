package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const CtxClerkUserID = "clerkUserId"

// InternalAuth production’da BFF’den gelen istekleri doğrular ve Clerk kullanıcı id’sini locals’a yazar.
// INTERNAL_API_SECRET boşsa (yerel Go-only deneme) atlanır; Clerk filtreleri de uygulanmaz.
func InternalAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if strings.TrimSpace(secret) == "" {
			return c.Next()
		}
		if c.Get("X-Internal-Secret") != secret {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		uid := strings.TrimSpace(c.Get("X-Clerk-User-Id"))
		if uid == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("missing X-Clerk-User-Id")
		}
		c.Locals(CtxClerkUserID, uid)
		return c.Next()
	}
}

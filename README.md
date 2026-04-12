# Expenses API (MVP)

Tek kullanıcı / tek liste senaryosu için harcama kayıtlarını saklayan REST API. **Sonraki aşama:** çoklu kullanıcı (`user_id`), isteğe bağlı **Next BFF** arkasında `INTERNAL_SECRET`, ileride **native istemci** için JWT (Clerk JWKS) doğrulama.

## Stack

| Katman | Seçim |
|--------|--------|
| Dil | **Go 1.22+** |
| HTTP | **[Fiber](https://gofiber.io/)** v2 |
| Veritabanı | **PostgreSQL** |
| ORM | **GORM** |

**Hosting:** Railway, Render, Fly.io vb. `Dockerfile` ile container deploy.

## API

| Method | Path | Açıklama |
|--------|------|----------|
| GET | `/health` | Sağlık kontrolü |
| GET | `/api/categories` | Kategoriler (ilk çalıştırmada seed) |
| GET | `/api/expenses` | Liste. Query: `from`, `to` — `YYYY-MM-DD` (MVP) |
| POST | `/api/expenses` | Yeni harcama |
| PATCH | `/api/expenses/:id` | Kısmi güncelleme (`amount`, `categoryId`, `note`, `occurredAt`) |
| DELETE | `/api/expenses/:id` | Sil (soft delete) |

**POST / PATCH gövdesi örneği:** `amount` (TL, ondalık), `categoryId`, `note?`, `occurredAt?` (`RFC3339` veya `YYYY-MM-DD`). Yanıtlarda `currency: "TRY"`; `userId` doluysa döner (şimdilik çoğunlukla `null`).

## Şema notu

- `expenses.user_id` — nullable, index; auth sonrası doldurulacak.

## Yerel geliştirme

```bash
docker compose up -d
cp .env.example .env
go run ./cmd/server
```

Varsayılan port **:8080**.

## Ortam değişkenleri

| Değişken | Zorunlu | Açıklama |
|----------|---------|----------|
| `DATABASE_URL` | evet | Postgres DSN |
| `PORT` | hayır | Varsayılan `8080` |
| `ALLOWED_ORIGINS` | hayır | CORS; BFF sonrası daraltılabilir (`*` yerine Vercel origin) |

**Planlanan (auth / BFF):** `INTERNAL_API_SECRET` — yalnızca Next sunucusunun Go’ya erişmesi için paylaşılan sır (dokümantasyon kod ile birlikte güncellenecek).

## Vercel

Bu servis **sürekli dinleyen** bir API’dir; tipik olarak **Render / Railway / Fly** üzerinde çalışır. Frontend Vercel’de kalabilir (şimdilik doğrudan URL; sonra BFF).

## Repo

Bu dizin bağımsız bir **git** deposu olabilir.

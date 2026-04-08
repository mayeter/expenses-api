# Expenses API (MVP)

Tek kullanıcı / tek liste senaryosu için harcama kayıtlarını saklayan REST API.

## Stack

| Katman | Seçim |
|--------|--------|
| Dil | **Go 1.22+** |
| HTTP | **[Fiber](https://gofiber.io/)** v2 |
| Veritabanı | **PostgreSQL** (ilişkisel; raporlama ve ileride çoklu kullanıcı için uygun) |
| ORM | **GORM** |

**Neden SQL / Postgres:** Harcama satırları ile kategoriler arasında net FK ilişkisi var; günlük ve kategori kırılımlı sorgular için Postgres yeterli ve bariz. NoSQL bu MVP için ek karmaşıklık sağlamaz.

**Hosting:** Railway, Render, Fly.io veya Supabase (Postgres connection string) ile uyumludur. İstersen kökteki `Dockerfile` ile container deploy edebilirsin.

## API

| Method | Path | Açıklama |
|--------|------|----------|
| GET | `/health` | Sağlık kontrolü |
| GET | `/api/categories` | Kategoriler (ilk çalıştırmada seed) |
| GET | `/api/expenses` | Liste. Query: `from`, `to` — `YYYY-MM-DD` (UTC günü, MVP) |
| POST | `/api/expenses` | Yeni harcama |
| DELETE | `/api/expenses/:id` | Sil (soft delete) |

**POST /api/expenses** gövdesi:

```json
{
  "amount": 149.99,
  "categoryId": 1,
  "note": "opsiyonel",
  "occurredAt": "2026-04-08T12:00:00Z"
}
```

- `amount`: **TL**, ondalıklı; sunucu kuruşa yuvarlar (`AmountMinor`).
- `occurredAt` yoksa: şu anki UTC zaman.
- `occurredAt` için `RFC3339` veya `YYYY-MM-DD` kabul edilir.

Yanıtlarda `currency` alanı şimdilik sabit `"TRY"`.

## Yerel geliştirme

1. Postgres’i ayağa kaldır:

```bash
docker compose up -d
```

2. Ortam dosyası:

```bash
cp .env.example .env
```

3. Çalıştır:

```bash
go run ./cmd/server
```

Sunucu varsayılan **:8080**.

## Ortam değişkenleri

| Değişken | Zorunlu | Açıklama |
|----------|---------|----------|
| `DATABASE_URL` | evet | Postgres DSN (`sslmode=require` Supabase vb. için) |
| `PORT` | hayır | Varsayılan `8080` |
| `ALLOWED_ORIGINS` | hayır | CORS; `*` veya `http://localhost:3000` gibi virgülle ayrılmış liste |

## Vercel notu

Bu proje klasik **sürekli çalışan HTTP sunucusu** modelindedir; Vercel’de Node serverless yerine **Railway / Render / Fly** daha doğrudan uyumludur. Frontend yine Vercel’de kalabilir; `NEXT_PUBLIC_API_URL` ile bu API’ye bağlanır.

## Repo

Bu dizin bağımsız bir **git** deposu olabilir.

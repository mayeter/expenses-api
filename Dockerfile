FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /server /server
ENV PORT=8080
EXPOSE 8080
USER nobody
CMD ["/server"]

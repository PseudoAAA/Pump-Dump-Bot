# ЭТАП 1: Сборка
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем всё остальное
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o bot-app main.go

# ЭТАП 2: Финальный образ
FROM alpine:latest

# Сертификаты и часовой пояс
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Europe/Moscow

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /app/bot-app .

# Копируем конфиги, соблюдая структуру
# Важно: если файла configbot.json нет в корне при сборке, Docker выдаст ошибку
COPY --from=builder /app/.env .
COPY --from=builder /app/configbot.json .

# Создаем структуру папок для внутренних конфигов
RUN mkdir -p internal/config
COPY --from=builder /app/internal/config/config.json ./internal/config/

# Создаем папку для графиков
RUN mkdir -p charts

# Запуск
CMD ["./bot-app"]
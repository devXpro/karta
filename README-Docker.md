# Karta Queue Monitor - Docker Setup

Этот проект использует Docker Compose с SurfShark VPN для работы из Польши.

## Требования

- Docker и Docker Compose
- SurfShark VPN аккаунт
- Telegram Bot Token

## Настройка

1. **Скопируйте файл окружения:**
   ```bash
   cp .env.example .env
   ```

2. **Заполните переменные в `.env`:**
   ```bash
   # Telegram Bot Token
   TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

   # SurfShark VPN Credentials
   SURFSHARK_USER=your_surfshark_email
   SURFSHARK_PASSWORD=your_surfshark_password
   ```

3. **Создайте директорию для данных:**
   ```bash
   mkdir -p data
   ```

## Запуск

```bash
# Сборка и запуск
docker compose up -d

# Просмотр логов
docker compose logs -f karta-bot

# Остановка
docker compose down
```

## Структура

- `karta-bot` - основное приложение
- `surfshark` - VPN контейнер для польского IP

## Мониторинг

```bash
# Статус контейнеров
docker compose ps

# Логи VPN
docker compose logs surfshark

# Логи бота
docker compose logs karta-bot

# Проверка IP (должен быть польский)
docker compose exec karta-bot wget -qO- ifconfig.me
```

## Troubleshooting

1. **Проблемы с VPN:**
   - Проверьте логи: `docker compose logs surfshark`
   - Убедитесь что учетные данные SurfShark правильные

2. **Проблемы с ботом:**
   - Проверьте логи: `docker compose logs karta-bot`
   - Убедитесь что Telegram Bot Token правильный

3. **Проблемы с сетью:**
   - Перезапустите контейнеры: `docker compose restart`
   - Проверьте IP: `docker compose exec karta-bot wget -qO- ifconfig.me`

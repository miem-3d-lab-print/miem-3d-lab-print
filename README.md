# MIEM 3D Lab Print

Веб-сервис лаборатории 3D-визуализации и компьютерной графики МИЭМ для подачи и обработки заявок на 3D-печать.

Пользователь входит по одноразовому коду из корпоративной почты, заполняет профиль, прикладывает модели STL/STEP/3MF или ZIP-архивы и следит за статусом заявки. Администратор управляет заявками, материалами, цветами, ролями и статистикой, а также может назначить конкретных администраторов получателями email-уведомлений о новых заявках.

## Быстрый запуск

Потребуется Docker Desktop или Docker Engine с Compose v2. Дополнительные SDK и реальная почта для локального запуска не нужны.

```bash
git clone https://github.com/miem-3d-lab-print/miem-3d-lab-print.git
cd miem-3d-lab-print
docker compose up --build -d
```

После запуска доступны:

| Компонент | Адрес |
| --- | --- |
| Приложение | <http://localhost:3000> |
| Проверка готовности | <http://localhost:3000/api/health> |
| Liveness | <http://localhost:3000/api/health/live> |
| OpenAPI YAML | <http://localhost:3000/api/openapi.yaml> |

Основной Compose публикует только TCP-порт `3000`. PostgreSQL, MinIO, Mailpit и backend доступны только контейнерам во внутренней Docker-сети. Для входа укажите адрес в домене `hse.ru`, `edu.hse.ru` или `miem.hse.ru`; при локальном запуске OTP отправляется во внутренний Mailpit.

Посмотреть список локальных писем, не открывая дополнительный порт:

```bash
docker compose exec backend wget -qO- http://mailpit:8025/api/v1/messages
```

Идентификатор нужного письма из поля `ID` можно использовать для получения полного содержимого:

```bash
docker compose exec backend wget -qO- http://mailpit:8025/api/v1/message/ID
```

Проверить состояние контейнеров и посмотреть логи:

```bash
docker compose ps
docker compose logs --since=1m -f
```

Остановить сервис:

```bash
docker compose down
```

Данные PostgreSQL и MinIO сохраняются в Docker volumes. Команда `docker compose down -v` удалит их без возможности восстановления и нужна только для полного сброса локального окружения.

## Развертывание на Ubuntu-сервере с IPv6

Этот вариант нужен, если хостинг блокирует исходящие SMTP-соединения по IPv4, но дает серверу полноценный IPv6. Docker Engine и Compose v2 должны быть уже установлены.

Создайте окружение и один раз задайте production-секреты и SMTP:

```bash
cd ~/miem-3d-lab-print
cp .env.example .env
nano .env
```

После этого вся настройка IPv6 и запуск выполняются одной командой:

```bash
make deploy-server
```

Скрипт `scripts/deploy-server.sh`:

- проверяет наличие IPv6-маршрута у хоста;
- сохраняет резервную копию `/etc/docker/daemon.json`;
- добавляет IPv6, не удаляя другие настройки Docker;
- проверяет `daemon.json` до перезапуска Docker;
- пересоздает Compose-сеть, собирает и запускает сервис;
- проверяет, что backend-контейнер получил IPv6.

Скрипт идемпотентен: его можно повторно запускать после `git pull`. Перезапуск Docker Engine выполняется только при изменении его конфигурации. Наружу по-прежнему публикуется только TCP-порт `3000`.

Для обновления и повторного развертывания на сервере достаточно одной строки:

```bash
cd ~/miem-3d-lab-print && git pull --ff-only && make deploy-server
```

## Настройка окружения

Сервис запускается с безопасными для локальной машины значениями по умолчанию. Чтобы изменить секреты, учетные данные или SMTP, создайте корневой `.env`:

```bash
cp .env.example .env
```

Основные переменные:

| Переменная | Значение по умолчанию | Назначение |
| --- | --- | --- |
| `JWT_SECRET` | development-only | Секрет JWT, минимум 32 символа |
| `DATABASE_URL` | локальный PostgreSQL в Compose | DSN backend |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | `25` / `10` | Размер connection pool GORM |
| `DB_CONN_MAX_LIFETIME` / `DB_CONN_MAX_IDLE_TIME` | `30m` / `5m` | Время жизни соединений |
| `POSTGRES_DB` / `POSTGRES_USER` / `POSTGRES_PASSWORD` | `miem3dprint` / `miem` / `miem` | Учетные данные PostgreSQL |
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | `minioadmin` / `minioadmin` | Учетные данные MinIO |
| `SMTP_HOST` / `SMTP_PORT` | `mailpit` / `1025` | SMTP-сервер |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | пусто | SMTP-аутентификация; задаются только парой |
| `SMTP_FROM` | `noreply@miem-3d-lab.local` | Адрес отправителя |

Публичный порт намеренно зафиксирован как `3000:80` в `docker-compose.yml`, чтобы вспомогательные сервисы нельзя было случайно открыть переменной окружения. Перед публичным развертыванием обязательно замените JWT-секрет, пароли PostgreSQL/MinIO и настройте настоящий SMTP. Если пароль PostgreSQL содержит специальные символы URL, передайте отдельно корректно URL-кодированный `DATABASE_URL`.

## Миграции

При `docker compose up` migrator выполняется до старта backend. Повторно применить все ещё не выполненные миграции:

```bash
make migrate
```

## Первый администратор

Сначала один раз войдите в приложение нужной учетной записью, затем назначьте роль:

```bash
docker compose exec db psql -U miem -d miem3dprint -c \
  "UPDATE users SET role = 'admin' WHERE lower(email) = lower('admin@edu.hse.ru');"
```

Если в `.env` изменены `POSTGRES_USER` или `POSTGRES_DB`, используйте новые значения в команде. После назначения роли выйдите и войдите снова, чтобы получить новый JWT.

## Локальная разработка

### Backend

Требования: Go 1.25 и Docker Compose.

Dev-compose в каталоге `backend` привязывает свои порты только к `127.0.0.1`. Они доступны на машине разработчика, но не принимают подключения с внешних сетевых интерфейсов.

```bash
cd backend
cp .env.example .env
docker compose up -d db minio mailpit
docker compose run --build --rm migrator
go run ./src/cmd/app
```

API будет доступен на <http://localhost:8080>, Mailpit — на <http://localhost:8025>.

### Frontend

Требования: Node.js 24 и npm 11.

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Frontend будет доступен на <http://localhost:5173> и обратится к backend по `http://localhost:8080/api`.

## Проверки качества

Настройки основаны на `avito-hackathon-8`: строгий TypeScript, ESLint для React и hooks, сортировка импортов, Stylelint, Prettier, `go vet`, race detector и golangci-lint с `errorlint`, `gocritic`, `misspell`, `gofmt` и `goimports`.

```bash
make test           # Go race tests и frontend Vitest
make lint           # Go и frontend линтеры в Docker
make check          # тесты, линтеры, vet и production-сборка frontend
```

Полезные отдельные команды:

```bash
cd frontend
npm run lint
npm run lint:styles
npm run format
npm test
npm run build

cd ../backend
go test ./...
go vet ./...
```

Те же проверки и сборка Docker-образов настроены в GitHub Actions: `.github/workflows/ci.yaml`.

## Архитектура

```text
miem-3d-lab-print/
├── backend/
│   ├── src/cmd/app/          # точка входа и сборка зависимостей
│   ├── src/internal/         # config, GORM repositories, services, handlers
│   ├── migrations/           # версионированный SQL migrator
│   └── docs/                 # сгенерированная OpenAPI/Swagger документация
├── frontend/
│   └── src/                  # React UI, API client, маршруты и типы
├── docker-compose.yml        # PostgreSQL, migrator, MinIO, Mailpit, backend и frontend
├── Makefile                  # единые команды запуска и проверок
├── API.md                    # описание HTTP API и бизнес-правил
└── DATABASE.md               # схема и инварианты базы данных
```

## Технологии

Основной стек приведён к подходам `avito-hackathon-8`:

- Go 1.25 и стандартный `net/http` с method-aware маршрутизацией;
- GORM + PostgreSQL 17, транзакции через `gorm.DB.Transaction` и настраиваемый connection pool;
- отдельный идемпотентный migrator с таблицей `schema_migrations` и advisory lock;
- JWT, bcrypt OTP, MinIO/S3 и SMTP;
- React 19, TypeScript, Vite, TanStack Query, SCSS, Vitest, Testing Library, ESLint, Stylelint и Prettier;
- Docker Compose, nginx и GitHub Actions.

Nginx — единственная публичная точка входа: он принимает запросы на порту `3000`, раздает SPA, ограничивает частоту OTP/API-запросов и проксирует `/api` в backend. Загруженные 3D-модели хранятся в MinIO, метаданные и история статусов — в PostgreSQL. GORM применяется для CRUD, фильтров и транзакций.

## Документация

- [API.md](./API.md) — эндпоинты, форматы запросов, ошибки и правила перехода статусов.
- [DATABASE.md](./DATABASE.md) — таблицы, связи, индексы и ограничения.
- OpenAPI YAML — `/api/openapi.yaml` у запущенного приложения.
- [frontend/README.md](./frontend/README.md) — детали frontend-разработки.

## Лицензия

См. [LICENSE](./LICENSE).

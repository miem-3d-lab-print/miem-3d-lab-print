# MIEM 3D Lab Print

Сервис подачи и обработки заявок на 3D-печать.

## Требования

- Docker Engine или Docker Desktop;
- Docker Compose v2;

## 1. Настройка `.env`

Создайте рабочий файл из примера:

```bash
cp .env.example .env
```

Перед публичным запуском обязательно измените:

```dotenv
JWT_SECRET=случайная_строка_не_короче_32_символов
POSTGRES_PASSWORD=надежный_пароль
MINIO_ROOT_USER=надежный_логин
MINIO_ROOT_PASSWORD=надежный_пароль
SITE_URL=http://91.200.150.116:3000
```

`DATABASE_URL` должен содержать те же имя пользователя, пароль и имя базы, что и `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`:

```dotenv
POSTGRES_DB=miem3dprint
POSTGRES_USER=miem
POSTGRES_PASSWORD=change-me
DATABASE_URL=postgres://miem:change-me@db:5432/miem3dprint?sslmode=disable
```

Если в пароле есть специальные символы, URL-кодируйте их внутри `DATABASE_URL`.

### Почта

Для локальной проверки оставьте Mailpit:

```dotenv
SMTP_HOST=mailpit
SMTP_PORT=1025
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=noreply@miem-3d-lab.local
```

Для рабочего SMTP укажите реальные параметры провайдера. Логин и пароль должны быть заполнены одновременно:

```dotenv
SMTP_HOST=smtp.example.ru
SMTP_PORT=587
SMTP_USERNAME=printer@example.ru
SMTP_PASSWORD=change-me
SMTP_FROM=printer@example.ru
```

`SITE_URL` используется в ссылках на заявки внутри писем.

## 2. Запуск

Соберите и запустите весь проект одной командой:

```bash
docker compose up --build -d
```

Миграции базы применяются автоматически контейнером `migrator` до запуска backend.

Проверить состояние:

```bash
docker compose ps
docker compose logs --since=5m backend frontend migrator
```

После запуска доступны:

- приложение: `http://localhost:3000`;
- проверка готовности: `http://localhost:3000/api/health`;
- OpenAPI YAML: `http://localhost:3000/swagger`.

## Первый администратор

Сначала войдите в приложение нужной корпоративной почтой, затем назначьте роль через контейнер PostgreSQL:

```bash
docker compose exec db psql -U miem -d miem3dprint -c \
  "UPDATE users SET role = 'admin' WHERE lower(email) = lower('admin@edu.hse.ru');"
```

После этого выйдите из приложения и войдите снова.

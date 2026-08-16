# API Backend части

**Base URL:** `http://localhost:8080/api` (dev) | `/api` (production за nginx)

**Аутентификация:** Bearer JWT в заголовке `Authorization: Bearer <access_token>`

**OpenAPI-спецификация:** [`backend/docs/swagger.yaml`](./backend/docs/swagger.yaml) | HTTP: `http://localhost:3000/swagger`

---

## Содержание

1. [Проверка готовности](#проверка-готовности)
2. [Формат ошибок](#формат-ошибок)
3. [Бизнес-правила](#бизнес-правила)
4. [Авторизация](#авторизация)
5. [Профиль](#профиль)
6. [Заявки — пользователь](#заявки-пользователь)
7. [Материалы — пользователь](#материалы-пользователь)
8. [Администратор — Заявки](#администратор--заявки)
9. [Администратор — Материалы](#администратор--материалы)
10. [Администратор — Пользователи](#администратор--пользователи)
11. [Администратор — Статистика](#администратор--статистика)
12. [Справочник кодов ошибок](#справочник-кодов-ошибок)

---

## Проверка готовности

### GET /health/live

Публичная liveness-проверка процесса. Не обращается к внешним зависимостям.

**Response 200:** `{ "service": "miem-3d-lab-print", "status": "alive" }`

### GET /health/ready

Публичный readiness endpoint. Проверяет доступность PostgreSQL.

`GET /health` является совместимым алиасом readiness endpoint.

**Response 200:** `{ "service": "miem-3d-lab-print", "status": "ready" }`

**Response 503:** `{ "service": "miem-3d-lab-print", "status": "not_ready" }`

---

## Формат ошибок

Все ошибки возвращаются в едином формате:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Человекочитаемое сообщение на русском",
    "details": {}
  }
}
```

Поле `details` опционально и используется для дополнительных данных (`attempts_left`, `locked_until`, `limit` и т.д.).

---

## Бизнес-правила

### Аутентификация и OTP

- **Допустимые домены email:** `hse.ru`, `edu.hse.ru`, `miem.hse.ru`. Любой другой домен отклоняется на этапе запроса кода.
- **Срок действия OTP:** 10 минут с момента отправки. Код хранится как bcrypt-хэш.
- **Лимит попыток:** 5 неверных попыток → OTP блокируется. `blocked_until` возвращается в `details`.
- **Rate limit по email:** не чаще 1 раза в 60 секунд. Засчитывается только успешная отправка письма.
- **Rate limit по IP:** не более 5 запросов в минуту. При наличии доверенного прокси берётся IP из правого поля `X-Forwarded-For`.
- **Refresh token rotation:** при обновлении старый токен немедленно отзывается и заменяется новым. Повторное использование отозванного токена (`REFRESH_TOKEN_REUSED`) отзывает всю цепочку — признак утечки.

### Профиль и согласие

- Согласие на обработку персональных данных даётся однократно. После этого поле `consent_given` не сбрасывается.
- Для подачи заявки профиль должен быть заполнен: `full_name` и хотя бы один контакт (`telegram` или `max`).
- Почти все эндпоинты, кроме `/auth/*` и `GET /profile`, требуют `consent_given = true`. При его отсутствии возвращается `403 CONSENT_REQUIRED`.

### Заявки

**Ограничения при создании:**
- Одновременно не более **10 активных** заявок на пользователя (статусы `new`, `in_review`, `printing`).
- Желаемая дата не может быть в прошлом.
- Если `color_matters = true` — `color_id` обязателен и должен принадлежать выбранному материалу.
- Выбранный материал и цвет должны быть активными (`is_active = true`).

**Файлы при создании:**
- Форматы: STL, STEP, 3MF, ZIP. Проверяется по расширению и сигнатуре байт.
- ZIP должен быть корректным архивом и содержать хотя бы одну валидную STL/STEP/STP/3MF-модель. До 100 элементов и до 200 МБ суммарного распакованного размера.
- Максимальный размер одного файла: **100 МБ**.
- Лимит файлов на заявку: **10 файлов** (суммарно, включая загруженные позже).

**Загрузка дополнительных файлов (`POST /applications/{id}/files`):**
- Доступна только пока статус заявки `new`.
- Действуют те же ограничения на формат, размер и лимит (10 файлов на заявку).

**Отмена:**
- Пользователь может отменить заявку только в статусе `new`.
- После отмены файлы хранятся ещё **7 дней**, затем удаляются из MinIO.

**Жизненный цикл файлов (TTL):**

| Событие | Срок хранения |
|---------|--------------|
| Заявка отменена пользователем | 7 дней |
| Заявка отклонена администратором | 7 дней |
| Заявка выдана (`issued`) | 30 дней |

**Снимки данных (snapshots):**
Имена материала, цвета и ФИО заявителя фиксируются в момент подачи заявки и не изменяются при последующем редактировании справочников или профиля. Поля `snapshot_material_name`, `snapshot_color_name`, `snapshot_full_name`, `snapshot_email` — исторически достоверны.

**Нумерация заявок:**
Формат: `YYYY-NNNN` (например, `2026-0042`). Используется отдельная PostgreSQL-последовательность на каждый год. Последовательности создаются при старте приложения для текущего и следующего года.

### Изменение статуса администратором

```
new → in_review → printing → ready → issued
                                    ↘ rejected  (обязателен rejection_reason)
cancelled — финальный, установлен пользователем, администратором не меняется
```

- Администратор может переводить заявку в любой из статусов: `in_review`, `printing`, `ready`, `issued`, `rejected`.
- Перевод в `rejected` требует непустого `rejection_reason`.
- Установка того же статуса (без смены) допустима только с непустым `comment` — используется для добавления комментария к истории.
- Заявка в статусе `cancelled` не может быть изменена.
- При смене статуса отправляется email уведомление заявителю.

### Материалы

- Деактивация материала (`is_active = false`) не удаляет его. Пользователь не может подать заявку на деактивированный материал, но уже поданные заявки с этим материалом остаются.
- Цвет может быть деактивирован независимо от материала.
- Имена материалов и цветов уникальны (регистронезависимо).

### Защита последнего администратора

- Нельзя снять роль `admin` с пользователя, если он единственный администратор в системе.
- Проверка выполняется внутри транзакции с блокировкой (`SELECT FOR UPDATE`) для защиты от гонок.

### Уведомления администраторов о заявках

- Email-уведомления о новых заявках включаются отдельно для каждого администратора.
- При снятии роли `admin` подписка на такие уведомления автоматически отключается.
- Почтовая ошибка не отменяет создание заявки и фиксируется в журнале backend.

---

## Авторизация

### POST /auth/request-otp

Запросить OTP-код на корпоративную почту.

**Доступ:** публичный

**Request:**
```json
{ "email": "ivanov@edu.hse.ru" }
```

**Response 200:**
```json
{ "message": "Код отправлен на почту", "expires_in": 600 }
```

| Код | HTTP | Условие |
|-----|------|---------|
| `INVALID_EMAIL_FORMAT` | 400 | Строка не является email |
| `INVALID_DOMAIN` | 400 | Домен не в списке допустимых |
| `RATE_LIMIT_EMAIL` | 429 | Повторный запрос раньше 60 сек |
| `RATE_LIMIT_IP` | 429 | Превышен лимит 5 запросов/мин с IP |
| `EMAIL_PROVIDER_ERROR` | 500 | SMTP-сервер недоступен |

---

### POST /auth/verify-otp

Подтвердить OTP-код и получить токены.

**Доступ:** публичный

**Request:**
```json
{ "email": "ivanov@edu.hse.ru", "code": "123456" }
```

**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 1800,
  "role": "user",
  "consent_required": true
}
```

`consent_required: true` — пользователь ещё не дал согласие на обработку ПД. Фронтенд должен перенаправить на страницу `/consent`.

| Код | HTTP | Условие |
|-----|------|---------|
| `OTP_NOT_FOUND` | 404 | Код не найден или уже использован |
| `CODE_EXPIRED` | 400 | Срок действия кода истёк |
| `INVALID_CODE` | 400 | Неверный код; `details.attempts_left` — осталось попыток |
| `LOCKED` | 423 | Исчерпаны попытки; `details.locked_until` — время разблокировки |

---

### POST /auth/refresh

Обновить access token и refresh token (rotation).

**Доступ:** публичный

**Request:**
```json
{ "refresh_token": "eyJ..." }
```

**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 1800
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `INVALID_REFRESH_TOKEN` | 401 | Токен невалиден |
| `REFRESH_TOKEN_EXPIRED` | 401 | Токен истёк |
| `REFRESH_TOKEN_REVOKED` | 401 | Токен отозван |
| `REFRESH_TOKEN_REUSED` | 401 | Повторное использование отозванного токена — вся цепочка отзывается |

---

### POST /auth/logout

Отозвать refresh token.

**Доступ:** публичный (передаётся refresh token в теле)

**Request:**
```json
{ "refresh_token": "eyJ..." }
```

**Response 204:** No Content

---

## Профиль

### GET /profile

Получить профиль текущего пользователя.

**Доступ:** авторизованный

**Response 200:**
```json
{
  "id": "uuid",
  "email": "ivanov@edu.hse.ru",
  "full_name": "Иванов Иван Иванович",
  "telegram": "@ivanov",
  "max": "ivanov_max",
  "role": "user",
  "consent_given": true,
  "consent_date": "2026-07-01T10:00:00Z",
  "created_at": "2026-06-01T00:00:00Z"
}
```

---

### PATCH /profile

Обновить профиль. Редактируются только `full_name`, `telegram`, `max`.

**Доступ:** авторизованный, согласие обязательно

**Request:**
```json
{
  "full_name": "Иванов Иван Иванович",
  "telegram": "@ivanov",
  "max": null
}
```

**Response 200:** Обновлённый профиль (см. GET /profile)

**Правила:** непустое `full_name` И хотя бы один контакт (`telegram` или `max`).

| Код | HTTP | Условие |
|-----|------|---------|
| `FIELD_NOT_EDITABLE` | 400 | Попытка изменить email |
| `INVALID_FULL_NAME` | 400 | ФИО пустое |
| `CONTACT_REQUIRED` | 400 | Не указан ни telegram, ни max |
| `CONSENT_REQUIRED` | 403 | Согласие не дано |

---

### POST /profile/consent

Дать согласие на обработку персональных данных (однократно, необратимо).

**Доступ:** авторизованный

**Response 200:**
```json
{ "consent_given": true, "consent_date": "2026-07-13T12:00:00Z" }
```

---

## Заявки (пользователь)

### GET /applications

Список своих заявок с пагинацией.

**Доступ:** авторизованный, согласие обязательно

**Query параметры:**

| Параметр | Тип | Описание |
|---------|-----|---------|
| `status` | string | Фильтр по статусу (`new`, `in_review`, `printing`, `ready`, `issued`, `rejected`, `cancelled`) |
| `page` | int | Страница (по умолчанию 1) |
| `per_page` | int | Размер страницы (1–100, по умолчанию 20) |

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "number": "2026-0001",
      "status": "new",
      "material_name": "PLA",
      "color_name": "Белый",
      "desired_date": "2026-08-01",
      "created_at": "2026-07-13T10:00:00Z",
      "files_count": 2
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 5, "total_pages": 1 }
}
```

---

### POST /applications/pending-files

Сразу загрузить выбранный файл новой заявки. Запрос `multipart/form-data`, поле `file`. В ответе возвращается UUID, который передаётся в `pending_file_ids[]` при создании заявки. Временный файл хранится 24 часа.

**Response 201:**
```json
{
  "id": "uuid",
  "filename": "model.stl",
  "size": 102400,
  "format": "STL",
  "created_at": "2026-08-16T10:00:00Z"
}
```

### DELETE /applications/pending-files/{file_id}

Удалить ранее загруженный временный файл, например когда пользователь убрал его из формы. Успешный ответ: `204 No Content`.

---

### POST /applications

Создать заявку. Запрос в формате `multipart/form-data`.

**Доступ:** авторизованный, согласие обязательно, заполненный профиль (`full_name` + `telegram` или `max`)

**Form fields:**

| Поле | Тип | Обязательное | Описание |
|------|-----|-------------|---------|
| `title` | string | да | Название заявки, до 255 символов |
| `position` | string | да | `bachelor`, `master`, `postgraduate`, `employee` |
| `purpose` | string | да | Цель печати |
| `material_id` | UUID | да | ID активного материала |
| `color_matters` | bool | да | Важен ли цвет (`true`/`false`) |
| `color_id` | UUID | если `color_matters=true` | ID активного цвета |
| `desired_date` | date (`YYYY-MM-DD`) | да | Желаемая дата (не в прошлом) |
| `comment` | string | нет | Комментарий |
| `file_url` | URL | если нет `pending_file_ids[]` | HTTP(S)-ссылка на файл или архив, до 2048 символов |
| `pending_file_ids[]` | UUID (multiple) | если нет `file_url` | Идентификаторы файлов, заранее загруженных через `POST /applications/pending-files` |

**Response 201:**
```json
{
  "id": "uuid",
  "number": "9b31c1ad-4912-4ced-b2d1-9f673ee4d719",
  "title": "Корпус датчика",
  "status": "new",
  "created_at": "2026-07-13T10:00:00Z"
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `PROFILE_NOT_FOUND` | 404 | Профиль не существует |
| `INVALID_APPLICATION_TITLE` | 422 | Название пустое или длиннее 255 символов |
| `INVALID_FILE_URL` | 422 | Ссылка не использует HTTP(S) или некорректна |
| `PENDING_FILE_NOT_FOUND` | 404 | Временный файл не найден, принадлежит другому пользователю или истёк |
| `PROFILE_INCOMPLETE` | 409 | Не заполнен `full_name` или не указан ни один контакт |
| `ACTIVE_LIMIT_REACHED` | 409 | Достигнут лимит в 10 активных заявок |
| `DESIRED_DATE_IN_PAST` | 400 | Желаемая дата в прошлом |
| `MATERIAL_NOT_FOUND` | 404 | Материал не найден |
| `MATERIAL_NOT_AVAILABLE` | 409 | Материал деактивирован |
| `COLOR_REQUIRED` | 400 | `color_matters=true`, но `color_id` не передан |
| `COLOR_NOT_FOUND` | 404 | Цвет не найден |
| `COLOR_NOT_AVAILABLE` | 409 | Цвет деактивирован |
| `INVALID_FILE_FORMAT` | 400 | Формат файла не STL/STEP/3MF/ZIP или ZIP не прошёл проверку |
| `FILE_TOO_LARGE` | 413 | Файл превышает 100 МБ |

---

### GET /applications/{id}

Получить детали своей заявки.

**Доступ:** авторизованный, согласие обязательно, только владелец заявки

**Response 200:**
```json
{
  "id": "uuid",
  "number": "2026-0001",
  "status": "in_review",
  "rejection_reason": null,
  "position": "bachelor",
  "purpose": "Учебный проект",
  "material": { "id": "uuid", "snapshot_name": "PLA" },
  "color_matters": true,
  "color": { "id": "uuid", "snapshot_name": "Белый" },
  "desired_date": "2026-08-01",
  "comment": null,
  "files": [
    { "id": "uuid", "filename": "model.stl", "size": 1048576, "format": "STL" }
  ],
  "files_delete_after": null,
  "status_history": [
    {
      "status": "new",
      "comment": null,
      "changed_by_role": "user",
      "created_at": "2026-07-13T10:00:00Z"
    }
  ],
  "created_at": "2026-07-13T10:00:00Z"
}
```

`files_delete_after` — дата удаления файлов (заполняется при отмене, отклонении или выдаче).

---

### PATCH /applications/{id}/cancel

Отменить заявку (только в статусе `new`).

**Доступ:** авторизованный, согласие обязательно, только владелец

**Response 200:**
```json
{
  "id": "uuid",
  "status": "cancelled",
  "files_delete_after": "2026-08-20T10:00:00Z"
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `CANCEL_NOT_ALLOWED` | 409 | Заявка не в статусе `new`; `details.current_status` |

---

### POST /applications/{id}/files

Загрузить дополнительный файл к заявке (только в статусе `new`).

**Доступ:** авторизованный, согласие обязательно, только владелец

**Form fields:**

| Поле | Описание |
|------|---------|
| `file` | Один файл STL/STEP/3MF/ZIP, до 100 МБ |

**Response 201:**
```json
{
  "id": "uuid",
  "filename": "model2.stl",
  "size": 512000,
  "format": "STL",
  "created_at": "2026-07-13T11:00:00Z"
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `FILES_REQUIRED` | 400 | Файл не передан |
| `INVALID_FILE_FORMAT` | 400 | Формат не поддерживается |
| `FILE_TOO_LARGE` | 413 | Файл > 100 МБ |
| `FILES_LOCKED` | 409 | Заявка не в статусе `new`; `details.current_status` |
| `FILES_LIMIT_REACHED` | 409 | Достигнут лимит в 10 файлов на заявку |

---

### GET /applications/{id}/files/{file_id}

Скачать файл. Backend потоково передаёт содержимое из объектного хранилища.

**Доступ:** авторизованный, согласие обязательно, только владелец

**Response 302:** `Location: https://minio/...?X-Amz-Signature=...`

| Код | HTTP | Условие |
|-----|------|---------|
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `FILE_NOT_FOUND` | 404 | Файл не найден |
| `FILE_DELETED` | 410 | Файл удалён по истечении срока хранения; `details.deleted_after` |

---

## Материалы (пользователь)

### GET /materials

Список доступных (активных) материалов с цветами.

**Доступ:** авторизованный, согласие обязательно

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "PLA",
      "description": "Жёсткий и экологичный пластик.",
      "colors": [
        { "id": "uuid", "name": "Белый" },
        { "id": "uuid", "name": "Чёрный" }
      ]
    }
  ]
}
```

Возвращаются только материалы с `is_active = true` и только их активные цвета.

---

## Администратор — Заявки

> Все эндпоинты `/admin/*` требуют роль `admin`.

### GET /admin/applications

Список всех заявок с фильтрацией и пагинацией.

**Query параметры:**

| Параметр | Тип | Описание |
|---------|-----|---------|
| `status` | string[] | Фильтр по статусам (можно несколько: `?status=new&status=in_review`) |
| `search` | string | Поиск по ФИО заявителя или номеру заявки |
| `material_id` | UUID | Фильтр по материалу |
| `created_from` | date | Начало диапазона даты создания (`YYYY-MM-DD`) |
| `created_to` | date | Конец диапазона (`YYYY-MM-DD`, включительно до 23:59:59) |
| `desired_from` | date | Начало диапазона желаемой даты |
| `desired_to` | date | Конец диапазона желаемой даты |
| `page` | int | Страница (по умолчанию 1) |
| `per_page` | int | Размер страницы (по умолчанию 20) |

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "number": "2026-0001",
      "full_name": "Иванов Иван Иванович",
      "created_at": "2026-07-13T10:00:00Z",
      "desired_date": "2026-08-01",
      "material_name": "PLA",
      "status": "in_review",
      "deadline_soon": false
    }
  ],
  "meta": { "page": 1, "per_page": 20, "total": 42, "total_pages": 3 }
}
```

`deadline_soon: true` — заявка в статусе `new` и желаемая дата наступает менее чем через 3 дня.

---

### GET /admin/applications/{id}

Детали заявки для администратора.

**Response 200:**
```json
{
  "id": "uuid",
  "number": "2026-0001",
  "applicant": {
    "user_id": "uuid",
    "snapshot_full_name": "Иванов Иван Иванович",
    "snapshot_email": "ivanov@edu.hse.ru",
    "telegram": "@ivanov",
    "max": null
  },
  "position": "bachelor",
  "purpose": "Учебный проект",
  "material": { "id": "uuid", "snapshot_name": "PLA", "is_active_now": true },
  "color_matters": true,
  "color": { "id": "uuid", "snapshot_name": "Белый" },
  "desired_date": "2026-08-01",
  "comment": null,
  "status": "in_review",
  "rejection_reason": null,
  "files": [
    { "id": "uuid", "filename": "model.stl", "size": 1048576, "format": "STL" }
  ],
  "status_history": [
    {
      "status": "new",
      "comment": null,
      "changed_by": { "id": "uuid", "full_name": "Иванов Иван", "role": "user" },
      "created_at": "2026-07-13T10:00:00Z"
    }
  ],
  "created_at": "2026-07-13T10:00:00Z"
}
```

`material.is_active_now` — актуальный статус материала на момент запроса (не snapshot).

---

### DELETE /admin/applications/{id}

Полностью удалить заявку. Доступно только администратору. Backend сначала удаляет все загруженные объекты заявки из MinIO, затем удаляет заявку, файлы и историю статусов из PostgreSQL.

Успешный ответ: `204 No Content`.

| Код | HTTP | Условие |
|-----|------|---------|
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `STORAGE_ERROR` | 502 | Не удалось удалить один из файлов из MinIO; запись заявки сохраняется |

---

### PATCH /admin/applications/{id}/status

Изменить статус заявки.

**Request:**
```json
{
  "status": "in_review",
  "comment": "Принято в работу",
  "rejection_reason": null
}
```

**Допустимые статусы для установки:** `in_review`, `printing`, `ready`, `issued`, `rejected`

**Правила:**
- При `status: "rejected"` поле `rejection_reason` обязательно (непустое).
- Установка того же статуса требует непустого `comment` — добавляет запись в историю без смены статуса.
- Заявка в статусе `cancelled` не изменяется.
- При успешной смене статуса на email заявителя уходит уведомление.

**Response 200:**
```json
{
  "id": "uuid",
  "status": "in_review",
  "files_delete_after": null
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `STATUS_NOT_ALLOWED` | 400 | Недопустимый статус |
| `REJECTION_REASON_REQUIRED` | 400 | Не указана причина отклонения |
| `COMMENT_REQUIRED` | 400 | Тот же статус, но без комментария |
| `APPLICATION_FINALIZED` | 409 | Заявка в статусе `cancelled` |

---

### GET /admin/applications/{id}/files/{file_id}

Скачать файл заявки. Backend потоково передаёт содержимое из объектного хранилища, не раскрывая внутренний адрес MinIO.

**Доступ:** авторизованный, роль `admin`

| Код | HTTP | Условие |
|-----|------|---------|
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `FILE_NOT_FOUND` | 404 | Файл не найден |
| `FILE_DELETED` | 410 | Файл удалён |

---

## Администратор — Материалы

### GET /admin/materials

Все материалы (включая неактивные) со всеми цветами.

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "name": "PLA",
      "description": "Жёсткий и экологичный пластик.",
      "is_active": true,
      "colors": [
        { "id": "uuid", "name": "Белый", "is_active": true }
      ]
    }
  ]
}
```

---

### POST /admin/materials

Создать материал.

**Request:**
```json
{ "name": "PETG", "description": "Прочный прозрачный пластик.", "is_active": true }
```

**Response 201:** Созданный материал (формат как в GET /admin/materials)

| Код | HTTP | Условие |
|-----|------|---------|
| `MATERIAL_NAME_EXISTS` | 409 | Материал с таким именем уже существует |

---

### PATCH /admin/materials/{id}

Обновить материал (название, описание, активность). Передавайте только изменяемые поля.

**Request:**
```json
{ "name": "PLA+", "description": null, "is_active": false }
```

**Response 200:** Обновлённый материал

| Код | HTTP | Условие |
|-----|------|---------|
| `MATERIAL_NOT_FOUND` | 404 | Материал не найден |
| `MATERIAL_NAME_EXISTS` | 409 | Имя занято другим материалом |

---

### POST /admin/materials/{id}/colors

Добавить цвет к материалу.

**Request:**
```json
{ "name": "Красный", "is_active": true }
```

**Response 201:**
```json
{ "id": "uuid", "name": "Красный", "is_active": true }
```

| Код | HTTP | Условие |
|-----|------|---------|
| `MATERIAL_NOT_FOUND` | 404 | Материал не найден |
| `COLOR_NAME_EXISTS` | 409 | Цвет с таким именем уже есть у материала |

---

### PATCH /admin/materials/{id}/colors/{color_id}

Обновить цвет (название, активность).

**Request:**
```json
{ "name": "Тёмно-красный", "is_active": false }
```

**Response 200:** Обновлённый цвет

| Код | HTTP | Условие |
|-----|------|---------|
| `MATERIAL_NOT_FOUND` | 404 | Материал не найден |
| `COLOR_NOT_FOUND` | 404 | Цвет не найден |
| `COLOR_NAME_EXISTS` | 409 | Имя занято другим цветом у этого материала |

---

## Администратор — Пользователи

### GET /admin/users

Поиск пользователей по email (минимум 3 символа).

**Query:** `?email=ivan`

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "email": "ivanov@edu.hse.ru",
      "full_name": "Иванов Иван Иванович",
      "role": "user",
      "application_notifications": false,
      "created_at": "2026-06-01T00:00:00Z"
    }
  ]
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `QUERY_TOO_SHORT` | 400 | Запрос короче 3 символов |

---

### PATCH /admin/users/{id}/role

Изменить роль пользователя.

**Request:**
```json
{ "role": "admin" }
```

Допустимые значения роли: `user`, `admin`.

**Response 200:**
```json
{ "id": "uuid", "email": "ivanov@edu.hse.ru", "role": "admin" }
```

| Код | HTTP | Условие |
|-----|------|---------|
| `INVALID_ROLE` | 400 | Значение не `user` и не `admin` |
| `USER_NOT_FOUND` | 404 | Пользователь не найден |
| `LAST_ADMIN` | 409 | Нельзя снять роль с единственного администратора |

---

### GET /admin/admins

Получить всех текущих администраторов. Список отсортирован по email.

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "email": "admin@edu.hse.ru",
      "full_name": "Иванов Иван Иванович",
      "role": "admin",
      "application_notifications": true,
      "created_at": "2026-06-01T00:00:00Z"
    }
  ]
}
```

---

### PATCH /admin/users/{id}/application-notifications

Включить или отключить email-уведомления администратора о новых заявках.

**Request:**
```json
{ "enabled": true }
```

**Response 200:**
```json
{
  "id": "uuid",
  "email": "admin@edu.hse.ru",
  "application_notifications": true
}
```

| Код | HTTP | Условие |
|-----|------|---------|
| `VALIDATION_ERROR` | 400 | Поле `enabled` отсутствует или не является boolean |
| `USER_NOT_FOUND` | 404 | Пользователь не найден |
| `USER_NOT_ADMIN` | 409 | Пользователь не имеет роль `admin` |

---

## Администратор — Статистика

### GET /admin/stats

Агрегированная статистика за период.

**Query:** `?date_from=2026-01-01&date_to=2026-12-31`

Если параметры не переданы, используется текущий календарный месяц.

**Response 200:**
```json
{
  "period": { "date_from": "2026-01-01", "date_to": "2026-12-31" },
  "total_applications": 150,
  "avg_completion_hours": 48.5,
  "completed_count": 120,
  "by_material": [
    { "material_name": "PLA", "count": 90 },
    { "material_name": "ABS", "count": 35 }
  ],
  "by_status_current": {
    "new": 5,
    "in_review": 10,
    "printing": 3,
    "ready": 2,
    "issued": 120,
    "rejected": 8,
    "cancelled": 2
  }
}
```

`by_status_current` — актуальное распределение всех заявок по статусам (без фильтрации по периоду).
`avg_completion_hours` — среднее время от создания заявки до перехода в `issued` за указанный период.

| Код | HTTP | Условие |
|-----|------|---------|
| `INVALID_PERIOD` | 400 | `date_from` позже `date_to` |

---

## Справочник кодов ошибок

| Код | HTTP | Описание |
|-----|------|---------|
| `INVALID_EMAIL_FORMAT` | 400 | Email в неверном формате |
| `INVALID_DOMAIN` | 400 | Домен не входит в список допустимых |
| `RATE_LIMIT_EMAIL` | 429 | Превышен rate limit по email |
| `RATE_LIMIT_IP` | 429 | Превышен rate limit по IP |
| `EMAIL_PROVIDER_ERROR` | 500 | Ошибка SMTP-провайдера |
| `OTP_NOT_FOUND` | 404 | OTP не найден или уже использован |
| `CODE_EXPIRED` | 400 | OTP истёк |
| `INVALID_CODE` | 400 | Неверный OTP; `details.attempts_left` |
| `LOCKED` | 423 | OTP заблокирован; `details.locked_until` |
| `INVALID_REFRESH_TOKEN` | 401 | Refresh token невалиден |
| `REFRESH_TOKEN_EXPIRED` | 401 | Refresh token истёк |
| `REFRESH_TOKEN_REVOKED` | 401 | Refresh token отозван |
| `REFRESH_TOKEN_REUSED` | 401 | Повторное использование отозванного токена |
| `UNAUTHORIZED` | 401 | Не аутентифицирован |
| `TOKEN_EXPIRED` | 401 | Access token истёк |
| `FORBIDDEN` | 403 | Недостаточно прав |
| `CONSENT_REQUIRED` | 403 | Требуется согласие на обработку ПД |
| `PROFILE_NOT_FOUND` | 404 | Профиль не найден |
| `FIELD_NOT_EDITABLE` | 400 | Поле нельзя редактировать |
| `INVALID_FULL_NAME` | 400 | Неверное ФИО |
| `CONTACT_REQUIRED` | 400 | Не указан контакт (telegram или max) |
| `PROFILE_INCOMPLETE` | 409 | Профиль не заполнен для подачи заявки |
| `APPLICATION_NOT_FOUND` | 404 | Заявка не найдена |
| `CANCEL_NOT_ALLOWED` | 409 | Отмена невозможна; `details.current_status` |
| `ACTIVE_LIMIT_REACHED` | 409 | Превышен лимит активных заявок (10) |
| `DESIRED_DATE_IN_PAST` | 400 | Желаемая дата в прошлом |
| `MATERIAL_NOT_FOUND` | 404 | Материал не найден |
| `MATERIAL_NOT_AVAILABLE` | 409 | Материал деактивирован |
| `MATERIAL_NAME_EXISTS` | 409 | Материал с таким именем уже существует |
| `COLOR_REQUIRED` | 400 | Необходимо указать цвет |
| `COLOR_NOT_FOUND` | 404 | Цвет не найден |
| `COLOR_NOT_AVAILABLE` | 409 | Цвет деактивирован |
| `COLOR_NAME_EXISTS` | 409 | Цвет с таким именем уже существует |
| `FILE_TOO_LARGE` | 413 | Файл превышает 100 МБ |
| `INVALID_FILE_FORMAT` | 400 | Неверный формат файла (не STL/STEP/3MF/ZIP) |
| `FILES_REQUIRED` | 400 | Файл не передан |
| `FILES_LOCKED` | 409 | Загрузка файлов недоступна (статус не `new`) |
| `FILES_LIMIT_REACHED` | 409 | Достигнут лимит в 10 файлов на заявку |
| `FILE_NOT_FOUND` | 404 | Файл не найден |
| `FILE_DELETED` | 410 | Файл удалён по истечении TTL; `details.deleted_after` |
| `STATUS_NOT_ALLOWED` | 400 | Статус недоступен для этой операции |
| `REJECTION_REASON_REQUIRED` | 400 | Требуется причина отклонения |
| `COMMENT_REQUIRED` | 400 | Требуется комментарий (при установке того же статуса) |
| `APPLICATION_FINALIZED` | 409 | Заявка в финальном статусе (`cancelled`) |
| `USER_NOT_FOUND` | 404 | Пользователь не найден |
| `INVALID_ROLE` | 400 | Недопустимая роль |
| `LAST_ADMIN` | 409 | Нельзя снять роль с последнего администратора |
| `QUERY_TOO_SHORT` | 400 | Запрос слишком короткий (< 3 символов) |
| `INVALID_PERIOD` | 400 | Некорректный период (`date_from` > `date_to`) |
| `STORAGE_ERROR` | 502 | Хранилище файлов (MinIO) недоступно |
| `INTERNAL_ERROR` | 500 | Внутренняя ошибка сервера |

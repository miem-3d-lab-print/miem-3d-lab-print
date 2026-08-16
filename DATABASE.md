# Модели базы данных

PostgreSQL 17 и GORM. SQL-миграции применяет отдельный контейнер `migrator`; выполненные версии фиксируются в `schema_migrations`, а конкурентный запуск защищён PostgreSQL advisory lock.

Файлы миграций: [`backend/migrations/`](./backend/migrations/)

---

## Содержание

1. [Диаграмма связей](#диаграмма-связей)
2. [ENUM-типы](#enum-типы)
3. [Таблицы](#таблицы)
   - [schema_migrations](#schema_migrations)
   - [users](#users)
   - [otp_codes](#otp_codes)
   - [refresh_tokens](#refresh_tokens)
   - [materials](#materials)
   - [colors](#colors)
   - [applications](#applications)
   - [files](#files)
   - [status_history](#status_history)
4. [Инварианты и ограничения](#инварианты-и-ограничения)
5. [Стратегия индексирования](#стратегия-индексирования)

---

## Диаграмма связей

```
users
 ├─── otp_codes          (по email, не FK)
 ├─── refresh_tokens     (user_id → users.id CASCADE DELETE)
 ├─── applications       (user_id → users.id RESTRICT DELETE)
 └─── status_history     (changed_by → users.id RESTRICT DELETE)

materials
 ├─── colors             (material_id → materials.id RESTRICT DELETE)
 └─── applications       (material_id → materials.id RESTRICT DELETE)

colors ──────────────── applications  (color_id → colors.id RESTRICT DELETE, nullable)

applications
 ├─── files              (application_id → applications.id CASCADE DELETE)
 └─── status_history     (application_id → applications.id CASCADE DELETE)
```

**Принципы:**
- `CASCADE DELETE` применяется только для дочерних данных, которые не имеют смысла без родителя (токены, файлы, история).
- `RESTRICT DELETE` защищает от случайного удаления пользователя или материала, на которые ссылаются заявки.
- `otp_codes` привязан к email без FK — пользователь может ещё не существовать в момент запроса OTP (первый вход).

---

## ENUM-типы

### `user_role`
```
'user'   — обычный пользователь (по умолчанию)
'admin'  — администратор
```

### `application_status`
```
'new'        — только создана, ожидает рассмотрения
'in_review'  — взята в работу администратором
'printing'   — печатается
'ready'      — готова к выдаче
'issued'     — выдана заявителю  [финальный + позитивный]
'rejected'   — отклонена         [финальный + негативный]
'cancelled'  — отменена пользователем (только из 'new') [финальный]
```

### `application_position`
```
'bachelor'      — бакалавр
'master'        — магистр
'postgraduate'  — аспирант
'employee'      — сотрудник
```

### `file_format`
```
'STL'   — STL (бинарный, ≥ 84 байта)
'STEP'  — STEP/STP (начинается с "ISO-10303-21")
'3MF'   — 3MF (ZIP-архив, сигнатура 0x504B)
'ZIP'   — ZIP-архив с одной или несколькими 3D-моделями
```

---

## Таблицы

### schema_migrations

Служебная таблица migrator. Хранит имя каждого успешно применённого SQL-файла и время применения; прикладной GORM-код её не изменяет.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `version` | VARCHAR(255) | NOT NULL | — | Имя migration-файла, PK |
| `applied_at` | TIMESTAMPTZ | NOT NULL | `now()` | Время успешного применения |

### users

Зарегистрированные пользователи. Создаётся при первом успешном входе (verify-otp).

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `email` | VARCHAR(255) | NOT NULL | — | Корпоративный email (lowercase-unique) |
| `full_name` | VARCHAR(255) | NULL | — | ФИО (заполняется в профиле) |
| `telegram` | VARCHAR(255) | NULL | — | Telegram-username |
| `max` | VARCHAR(255) | NULL | — | Логин в системе MAX |
| `telegram_id` | BIGINT | NULL | — | Telegram user ID (для бота) |
| `role` | `user_role` | NOT NULL | `'user'` | Роль |
| `consent_given` | BOOLEAN | NOT NULL | `false` | Дано ли согласие на обработку ПД |
| `consent_given_at` | TIMESTAMPTZ | NULL | — | Дата согласия |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата регистрации |
| `updated_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата обновления |

**Ограничения:**
- `users_consent_check` — если `consent_given = true`, то `consent_given_at IS NOT NULL`.

**Уникальные индексы:**
- `users_email_key` — `lower(email)` (регистронезависимая уникальность)
- `users_telegram_id_key` — `telegram_id WHERE telegram_id IS NOT NULL`

---

### otp_codes

Одноразовые коды для входа. Не привязан FK к `users` — код можно запросить до первой регистрации.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `email` | VARCHAR(255) | NOT NULL | — | Email, для которого выдан код |
| `code_hash` | VARCHAR(60) | NOT NULL | — | bcrypt-хэш 6-значного кода |
| `attempts` | SMALLINT | NOT NULL | `0` | Количество неверных попыток (0–5) |
| `expires_at` | TIMESTAMPTZ | NOT NULL | — | Срок действия (10 минут с создания) |
| `blocked_until` | TIMESTAMPTZ | NULL | — | Блокировка после 5 попыток |
| `is_used` | BOOLEAN | NOT NULL | `false` | Код уже использован |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата создания |

**Индексы:**
- `otp_codes_email_idx` — `email WHERE is_used = false` (частичный, поиск активных кодов)
- `otp_codes_cleanup_idx` — `expires_at` (для периодической очистки устаревших записей)

---

### refresh_tokens

Refresh-токены с поддержкой rotation и обнаружения повторного использования.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `user_id` | UUID | NOT NULL | — | FK → `users.id` CASCADE DELETE |
| `token_hash` | VARCHAR(64) | NOT NULL | — | SHA-256 хэш токена (уникальный) |
| `expires_at` | TIMESTAMPTZ | NOT NULL | — | Срок действия |
| `revoked_at` | TIMESTAMPTZ | NULL | — | Дата отзыва (NULL = активен) |
| `replaced_by_token_id` | UUID | NULL | — | FK → `refresh_tokens.id` (цепочка ротации) |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата выдачи |

**Логика rotation:** при обновлении старый токен помечается `revoked_at = now()`, `replaced_by_token_id = <id нового>`. Попытка использовать отозванный токен — сигнал компрометации, вся цепочка отзывается.

---

### materials

Каталог материалов для 3D-печати.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `name` | VARCHAR(100) | NOT NULL | — | Название (lowercase-unique) |
| `description` | TEXT | NOT NULL | `''` | Описание |
| `is_active` | BOOLEAN | NOT NULL | `true` | Доступен для выбора в заявках |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата создания |

Начальные данные (seed): `PLA`, `ABS`, `TPU`.

---

### colors

Цвета, привязанные к конкретному материалу.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `material_id` | UUID | NOT NULL | — | FK → `materials.id` RESTRICT DELETE |
| `name` | VARCHAR(100) | NOT NULL | — | Название цвета |
| `is_active` | BOOLEAN | NOT NULL | `true` | Доступен для выбора |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата создания |

**Уникальный индекс:** `colors_material_name_key` — `(material_id, lower(name))` — имя уникально в рамках материала.

---

### applications

Заявки на 3D-печать. Центральная сущность системы.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `number` | VARCHAR(12) | NOT NULL | — | Уникальный номер `YYYY-NNNN` |
| `title` | VARCHAR(255) | NOT NULL | `'Заявка на 3D-печать'` | Название заявки |
| `user_id` | UUID | NOT NULL | — | FK → `users.id` RESTRICT DELETE |
| `snapshot_full_name` | VARCHAR(255) | NOT NULL | — | ФИО заявителя на момент подачи |
| `snapshot_email` | VARCHAR(255) | NOT NULL | — | Email заявителя на момент подачи |
| `position` | `application_position` | NOT NULL | — | Позиция заявителя |
| `purpose` | TEXT | NOT NULL | — | Цель печати |
| `material_id` | UUID | NOT NULL | — | FK → `materials.id` RESTRICT DELETE |
| `snapshot_material_name` | VARCHAR(100) | NOT NULL | — | Название материала на момент подачи |
| `color_matters` | BOOLEAN | NOT NULL | — | Важен ли цвет |
| `color_id` | UUID | NULL | — | FK → `colors.id` RESTRICT DELETE |
| `snapshot_color_name` | VARCHAR(100) | NULL | — | Название цвета на момент подачи |
| `desired_date` | DATE | NOT NULL | — | Желаемая дата получения |
| `comment` | TEXT | NULL | — | Комментарий заявителя |
| `file_url` | TEXT | NULL | — | HTTP(S)-ссылка на файл вместо загрузки |
| `status` | `application_status` | NOT NULL | `'new'` | Текущий статус |
| `rejection_reason` | TEXT | NULL | — | Причина отклонения (только для `rejected`) |
| `files_delete_after` | TIMESTAMPTZ | NULL | — | Дата удаления файлов (TTL) |
| `deadline_notified` | BOOLEAN | NOT NULL | `false` | Уведомление о приближении срока отправлено |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата создания |

**Ограничения:**
- `color_consistency` — если `color_matters = false`, то `color_id IS NULL` и `snapshot_color_name IS NULL`; если `true` — оба поля заполнены.
- `rejection_reason_check` — если `status = 'rejected'`, то `rejection_reason IS NOT NULL`.

**Snapshot-поля** (`snapshot_*`) — неизменяемые снимки данных на момент подачи заявки. Изменение материала, цвета или профиля пользователя после подачи не влияет на историю.

**Нумерация:** используется PostgreSQL-последовательность `app_number_{year}` (создаётся при старте приложения для текущего и следующего года).

---

### files

Прикреплённые 3D-файлы. Физически хранятся в MinIO, в таблице — метаданные и путь.

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `application_id` | UUID | NOT NULL | — | FK → `applications.id` CASCADE DELETE |
| `filename` | VARCHAR(255) | NOT NULL | — | Оригинальное имя файла |
| `storage_path` | VARCHAR(512) | NOT NULL | — | Путь в MinIO (`applications/{app_id}/{file_id}.ext`) |
| `size` | INTEGER | NOT NULL | — | Размер в байтах (1 — 20 971 520) |
| `format` | `file_format` | NOT NULL | — | Формат файла |
| `deleted_at` | TIMESTAMPTZ | NULL | — | Дата фактического удаления из MinIO |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата загрузки |

`deleted_at IS NOT NULL` означает, что файл физически удалён из MinIO. Запись в БД при этом сохраняется для истории.

---

### status_history

Полная история изменений статуса заявки. Записи только добавляются, никогда не удаляются (каскадно удаляются только вместе с заявкой).

| Колонка | Тип | Nullable | По умолчанию | Описание |
|---------|-----|----------|-------------|---------|
| `id` | UUID | NOT NULL | `gen_random_uuid()` | PK |
| `application_id` | UUID | NOT NULL | — | FK → `applications.id` CASCADE DELETE |
| `status` | `application_status` | NOT NULL | — | Статус, в который перешла заявка |
| `comment` | TEXT | NULL | — | Комментарий к переходу |
| `changed_by` | UUID | NOT NULL | — | FK → `users.id` RESTRICT DELETE |
| `created_at` | TIMESTAMPTZ | NOT NULL | `now()` | Дата перехода |

Первая запись создаётся автоматически при создании заявки: `status = 'new'`, `changed_by = user_id`.

---

## Инварианты и ограничения

| Инвариант | Где проверяется |
|-----------|----------------|
| Email уникален (без учёта регистра) | Partial unique index |
| Цвет уникален в рамках материала | Partial unique index |
| Путь файла в MinIO уникален | Unique index |
| Цвет согласован с `color_matters` | CHECK `color_consistency` |
| `rejection_reason` обязателен для `rejected` | CHECK `rejection_reason_check` |
| `consent_given_at` заполнен при `consent_given=true` | CHECK `users_consent_check` |
| Размер файла: 1–20 МБ | CHECK `size > 0 AND size <= 20971520` |
| Не более 5 попыток OTP | CHECK `attempts BETWEEN 0 AND 5` |
| Нельзя удалить последнего admin | Транзакция + SELECT FOR UPDATE в сервисе |
| Не более 10 активных заявок на пользователя | Проверка в сервисе внутри транзакции |
| Не более 10 файлов на заявку | Проверка в сервисе до загрузки |

---

## Стратегия индексирования

| Индекс | Тип | Назначение |
|--------|-----|-----------|
| `users_email_key` | Unique partial | Поиск пользователя при входе |
| `users_role_admin_idx` | Partial | Быстрый подсчёт администраторов |
| `otp_codes_email_idx` | Partial (`is_used = false`) | Поиск активного OTP по email |
| `otp_codes_cleanup_idx` | B-tree | Периодическая очистка устаревших OTP |
| `refresh_tokens_hash_key` | Unique | Верификация refresh token |
| `refresh_tokens_user_idx` | B-tree | Отзыв всех токенов пользователя |
| `refresh_tokens_cleanup_idx` | B-tree | Очистка истёкших токенов |
| `materials_name_key` | Unique partial | Уникальность имени материала |
| `colors_material_name_key` | Unique partial | Уникальность имени цвета в материале |
| `applications_number_key` | Unique | Уникальность номера заявки |
| `applications_user_idx` | B-tree | Список заявок пользователя |
| `applications_created_idx` | B-tree DESC | Сортировка по дате создания |
| `applications_status_idx` | B-tree | Фильтрация по статусу |
| `applications_active_per_user_idx` | Partial | Подсчёт активных заявок (лимит 10) |
| `applications_deadline_idx` | Partial (`status = 'new'`) | Уведомления о приближающемся сроке |
| `applications_files_ttl_idx` | Partial | Задание по удалению устаревших файлов |
| `files_application_idx` | B-tree | Список файлов заявки |
| `files_storage_path_key` | Unique | Уникальность пути в MinIO |
| `status_history_app_idx` | B-tree composite | История статусов заявки |
| `status_history_status_created_idx` | B-tree composite | Поиск перехода в `issued` (для avg_completion_hours) |

# Frontend MIEM 3D Lab Print

React 19 + TypeScript + Vite frontend сервиса заявок на 3D-печать. Стили собираются из SCSS, unit-тесты выполняются через Vitest и Testing Library.

## Запуск

```bash
cp .env.example .env
npm install
npm run dev
```

Dev-сервер: <http://localhost:5173>. По умолчанию запросы отправляются в `http://localhost:8080/api`.

## Переменные

| Переменная | По умолчанию | Назначение |
| --- | --- | --- |
| `VITE_API_URL` | `http://localhost:8080/api` | Базовый URL backend API |
| `VITE_FILES_FIELD_NAME` | `files[]` | Multipart-поле загружаемых моделей |

Переменные Vite встраиваются во frontend во время сборки. Для Docker Compose уже заданы `VITE_API_URL=/api` и nginx-прокси.

В полном Docker Compose frontend собирается в nginx-контейнер и доступен на <http://localhost:3000>. Это единственный опубликованный порт основного стека.

## Команды

```bash
npm run dev
npm run build
npm run preview
npm run lint
npm run lint:fix
npm run lint:styles
npm run lint:styles:fix
npm run format
npm run format:check
npm test
npm run test:watch
```

Общие правила и команды проверки всего репозитория описаны в [корневом README](../README.md).

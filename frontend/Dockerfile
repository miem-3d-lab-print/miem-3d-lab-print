FROM node:24-alpine AS build

WORKDIR /app

COPY package*.json ./
RUN npm install --no-audit --no-fund

COPY . .

ARG VITE_API_URL=/api
ARG VITE_FILES_FIELD_NAME=files[]
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_FILES_FIELD_NAME=$VITE_FILES_FIELD_NAME
RUN npm run build

FROM nginx:1.29-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

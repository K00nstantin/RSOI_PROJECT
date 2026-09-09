FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:alpine
# Копируем собранную статику
COPY --from=builder /app/build /usr/share/nginx/html
# Копируем кастомный конфиг Nginx (переопределяет дефолтный)
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]



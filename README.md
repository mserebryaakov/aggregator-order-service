## order-service - golang сервис для работы с заказами и доставками

### Краткое описание:

- Gin
- Postgre
- Gorm
- Docker (docker-compose)
- Swagger 3.0

### Применены следующие пакеты:

- gin - http framework
- viper - конфигурация
- logrus - логирование
- godotenv - переменные окружения
- gorm - orm библиотека (postgre driver)

### Развертывание с помощью dockerfile:

- Убедиться, что в файле конфига `config/config.json` имя хоста в конфигурации Minio - `"host": "localhost"`
- Убедиться, что Minio запущена локально
- Выполнить `docker build . -t file-service:latest`
- Выполнить `docker run --env-file ./.env -p 8080:8080 file-service:latest`

### Развёртывание с помощью docker-compose:

Для запуска сервиса с помощью команды `docker-compose up` необходимо:
- Убедиться, что в файле конфига `config/config.json` имя хоста в конфигурации Minio совпадает с названием сервиса в файле `docker-compose.yml` (по умолчанию - `"host": "miniodb"`).
- Убедиться, что 8080, 9000 и 9090 порты не заняты
- Выполнить команду `docker-compose up`.
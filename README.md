# SubPub - cистема публикации и подписки на события

## 📋 Содержание

- [Описание](#описание)
- [Архитектура](#архитектура)
- [Установка](#установка)
- [Запуск](#запуск)
- [Тестирование](#тестирование)
- [API](#api)
- [Конфигурация](#конфигурация)
- [Паттерны проектирования](#паттерны-проектирования)
- [Логирование](#логирование)
- [Примеры использования](#примеры-использования)

## 📝 Описание

SubPub - это высокопроизводительная система публикации и подписки на события, реализованная на Go с использованием gRPC. Проект разработан с фокусом на надежность, масштабируемость и простоту использования.

Система позволяет клиентам:
- Подписываться на события по любому ключу
- Получать поток сообщений в режиме реального времени
- Публиковать сообщения для всех подписчиков определенного ключа

## 🏗 Архитектура

Проект построен в соответствии с принципами чистой архитектуры:

```
├── cmd/                 # Исполняемые файлы
│   ├── server/          # gRPC сервер
│   └── client/          # Пример клиента
├── internal/            # Внутренние пакеты приложения
│   ├── config/          # Конфигурация
│   ├── server/          # Реализация gRPC сервера
│   └── service/         # Бизнес-логика сервиса
├── proto/               # Protobuf определения
└── subpub/              # Основная библиотека pub/sub механизма
```

## 🛠 Установка

### Предварительные требования

- Go 1.18+
- Git

### Клонирование и установка зависимостей

```bash
# Клонирование репозитория
git clone https://github.com/Jonnydevp/subpub.git
cd subpub

# Установка зависимостей
go mod download
go mod tidy
```

## 🚀 Запуск

### Компиляция

```bash
# Сборка сервера
go build -o bin/server cmd/server/main.go

# Сборка клиента
go build -o bin/client cmd/client/main.go
```

### Запуск сервера

```bash
# Запуск с настройками по умолчанию
./bin/server

# Или напрямую из исходников
go run cmd/server/main.go
```

### Запуск клиента

```bash
# Запуск скомпилированного клиента
./bin/client

# Или напрямую из исходников
go run cmd/client/main.go
```

### Быстрая демонстрация

1. **Запустите сервер** в первом терминале:
   ```bash
   cd D:\gopubsub
   go run cmd/server/main.go
   ```

2. **Запустите клиент** во втором терминале:
   ```bash
   cd D:\gopubsub
   go run cmd/client/main.go
   ```

3. **Наблюдайте за логами** в обоих терминалах:
   - В терминале сервера вы увидите информацию о новой подписке и публикации
   Примеры логов:
```json
{"level":"info","ts":1620000000,"msg":"New subscription","key":"test"}
{"level":"info","ts":1620000001,"msg":"Message published","key":"test"}
```
   - В терминале клиента вы увидите полученное сообщение
     // Даем время на получение сообщения
    time.Sleep(3 * time.Second)
    log.Println("2023/12/01 15:00:00 Получено: Сообщение от клиента")

```

## 🧪 Тестирование

### Запуск тестов

```bash
# Запуск всех тестов
go test ./...

# Тестирование основного пакета subpub
go test ./subpub

# Запуск с подробным выводом
go test -v ./subpub

# Запуск с покрытием кода
go test -cover ./subpub
```


## 📡 API

### gRPC API

Система предоставляет два основных метода:

1. **Subscribe** - Потоковый метод для подписки на события
   ```protobuf
   rpc Subscribe(SubscribeRequest) returns (stream Event);
   ```

2. **Publish** - Метод для публикации событий
   ```protobuf
   rpc Publish(PublishRequest) returns (google.protobuf.Empty);
   ```

### Пример использования API

```go
// Создание клиента
conn, _ := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
client := pb.NewPubSubClient(conn)

// Подписка на события
stream, _ := client.Subscribe(context.Background(), &pb.SubscribeRequest{Key: "notifications"})

// Чтение потока событий
go func() {
    for {
        event, err := stream.Recv()
        if (err != nil) {
            break
        }
        fmt.Printf("Получено событие: %s\n", event.Data)
    }
}()

// Публикация события
client.Publish(context.Background(), &pb.PublishRequest{
    Key:  "notifications",
    Data: "Важное уведомление",
})
```

## ⚙️ Конфигурация

Приложение поддерживает настройку через:

1. **Файл конфигурации YAML**:
   ```yaml
   grpc:
     addr: ":50051"
   log:
     level: "info"
   shutdown_timeout: 10s
   ```

## 🧩 Паттерны проектирования

В проекте применены современные паттерны и практики разработки:

- **Dependency Injection** - все зависимости явно передаются через конструкторы
- **Graceful Shutdown** - корректное завершение работы сервера и освобождение ресурсов
- **Middleware** (Interceptors) - для централизованной обработки и логирования запросов
- **Repository Pattern** - для хранения и управления подписками
- **Observer Pattern** - в основе механизма публикации/подписки
- **Context Pattern** - для управления временем жизни запросов и отмены операций

## 📊 Логирование

Система использует структурированное логирование с помощью библиотеки `zap`:

- **Уровни логирования**: debug, info, warn, error
- **Структурированные логи** в формате JSON для удобной обработки
- **Контекстные метаданные** (ключ события, ID запроса и пр.)
- **Производительность** - минимальное влияние на быстродействие системы



## 📚 Примеры использования

### Базовый пример

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/Jonnydevp/subpub/proto"
)

func main() {
    conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatal("Ошибка подключения:", err)
    }
    defer conn.Close()

    client := pb.NewPubSubClient(conn)

    // Подписка на канал "test"
    stream, err := client.Subscribe(context.Background(), &pb.SubscribeRequest{Key: "test"})
    if err != nil {
        log.Fatal("Ошибка подписки:", err)
    }

    // Горутина для получения сообщений
    go func() {
        for {
            msg, err := stream.Recv()
            if err != nil {
                log.Fatal("Ошибка получения:", err)
            }
            log.Printf("Получено: %s", msg.GetData())
        }
    }()

    // Публикация сообщения
    _, err = client.Publish(context.Background(), &pb.PublishRequest{
        Key:  "test",
        Data: "Сообщение от клиента",
    })
    if err != nil {
        log.Fatal("Ошибка публикации:", err)
    }


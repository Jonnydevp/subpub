package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscribeAndPublish(t *testing.T) {
	sp := NewSubPub()

	// Тестовые данные
	subject := "test-subject"
	msgs := []string{"message1", "message2", "message3"}

	// Канал для сбора результатов
	resultChan := make(chan string, len(msgs)*2)

	// Подписываем двух подписчиков
	cb1 := func(msg interface{}) {
		resultChan <- "sub1:" + msg.(string)
	}

	cb2 := func(msg interface{}) {
		resultChan <- "sub2:" + msg.(string)
	}

	sub1, err := sp.Subscribe(subject, cb1)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	_, err = sp.Subscribe(subject, cb2)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Публикуем сообщения
	for _, msg := range msgs {
		if err := sp.Publish(subject, msg); err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Собираем результаты
	receivedMsgs := make([]string, 0, len(msgs)*2)
	timeout := time.After(1 * time.Second)

	for i := 0; i < len(msgs)*2; i++ {
		select {
		case msg := <-resultChan:
			receivedMsgs = append(receivedMsgs, msg)
		case <-timeout:
			t.Fatalf("Timeout waiting for messages. Got %d out of %d expected", len(receivedMsgs), len(msgs)*2)
		}
	}

	// Проверка: количество сообщений должно быть равно количеству отправленных * количество подписчиков
	if len(receivedMsgs) != len(msgs)*2 {
		t.Errorf("Expected %d messages, got %d", len(msgs)*2, len(receivedMsgs))
	}

	// Проверяем отписку
	sub1.Unsubscribe()

	// Публикуем еще одно сообщение
	if err := sp.Publish(subject, "after-unsubscribe"); err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Должно прийти только одно сообщение (для sub2)
	select {
	case msg := <-resultChan:
		if msg != "sub2:after-unsubscribe" {
			t.Errorf("Expected message from sub2, got %s", msg)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message after unsubscribe")
	}

	// Проверяем, что больше сообщений не приходит
	select {
	case msg := <-resultChan:
		t.Errorf("Unexpected message received: %s", msg)
	case <-time.After(100 * time.Millisecond):
		// Это ожидаемое поведение
	}

	// Закрываем систему
	ctx := context.Background()
	if err := sp.Close(ctx); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

func TestFIFOOrder(t *testing.T) {
	sp := NewSubPub()
	subject := "fifo-test"

	// Канал для результатов
	results := make([]int, 0, 10)
	var mu sync.Mutex

	// Колбэк, который добавляет сообщения в результат
	cb := func(msg interface{}) {
		mu.Lock()
		results = append(results, msg.(int))
		mu.Unlock()
	}

	// Подписываемся
	_, err := sp.Subscribe(subject, cb)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Публикуем сообщения
	for i := 0; i < 10; i++ {
		if err := sp.Publish(subject, i); err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Ждем обработки всех сообщений
	time.Sleep(100 * time.Millisecond)

	// Проверяем порядок сообщений
	mu.Lock()
	defer mu.Unlock()

	if len(results) != 10 {
		t.Fatalf("Expected 10 messages, got %d", len(results))
	}

	for i, v := range results {
		if i != v {
			t.Errorf("Messages received out of order. Expected %d at position %d, got %d", i, i, v)
		}
	}

	// Закрываем систему
	ctx := context.Background()
	if err := sp.Close(ctx); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

func TestSlowSubscriber(t *testing.T) {
	sp := NewSubPub()
	subject := "slow-test"

	// Канал для результатов быстрого подписчика
	fastResults := make(chan string, 10)

	// Канал для результатов медленного подписчика
	slowResults := make(chan string, 10)

	// Быстрый обработчик
	fastCb := func(msg interface{}) {
		fastResults <- msg.(string)
	}

	// Медленный обработчик
	slowCb := func(msg interface{}) {
		// Имитируем медленную обработку
		time.Sleep(100 * time.Millisecond)
		slowResults <- msg.(string)
	}

	// Подписываемся
	_, err := sp.Subscribe(subject, fastCb)
	if err != nil {
		t.Fatalf("Failed to subscribe fast handler: %v", err)
	}

	_, err = sp.Subscribe(subject, slowCb)
	if err != nil {
		t.Fatalf("Failed to subscribe slow handler: %v", err)
	}

	// Публикуем сообщения
	messages := []string{"msg1", "msg2", "msg3"}
	for _, msg := range messages {
		if err := sp.Publish(subject, msg); err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Проверяем, что быстрый подписчик получил все сообщения быстро
	timeout := time.After(50 * time.Millisecond)
	receivedFast := make([]string, 0, len(messages))

collectFast:
	for i := 0; i < len(messages); i++ {
		select {
		case msg := <-fastResults:
			receivedFast = append(receivedFast, msg)
		case <-timeout:
			break collectFast
		}
	}

	// Проверяем, что быстрый подписчик получил все сообщения
	if len(receivedFast) != len(messages) {
		t.Errorf("Fast subscriber got %d messages, expected %d", len(receivedFast), len(messages))
	}

	// Ждем, чтобы дать медленному подписчику время обработать все сообщения
	time.Sleep(500 * time.Millisecond)

	// Проверяем, что медленный подписчик тоже получил все сообщения
	receivedSlow := make([]string, 0, len(messages))
	for i := 0; i < len(messages); i++ {
		select {
		case msg := <-slowResults:
			receivedSlow = append(receivedSlow, msg)
		default:
			t.Fatalf("Slow subscriber did not receive all messages. Got %d, expected %d", len(receivedSlow), len(messages))
		}
	}

	// Закрываем систему
	ctx := context.Background()
	if err := sp.Close(ctx); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

func TestCloseWithContext(t *testing.T) {
	sp := NewSubPub()
	subject := "close-test"

	// Создаем медленный обработчик, который блокирует выполнение
	done := make(chan struct{})
	cb := func(msg interface{}) {
		<-done // Блокируем обработчик
	}

	// Подписываемся
	_, err := sp.Subscribe(subject, cb)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Публикуем сообщение
	if err := sp.Publish(subject, "test"); err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Закрываем систему с контекстом
	start := time.Now()
	err = sp.Close(ctx)
	elapsed := time.Since(start)

	// Проверяем, что метод Close вернул ошибку контекста
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got: %v", err)
	}

	// Проверяем, что Close не блокировался дольше таймаута контекста
	if elapsed > 150*time.Millisecond {
		t.Errorf("Close took too long: %v, expected around 100ms", elapsed)
	}

	// Закрываем канал done, чтобы разблокировать обработчик
	close(done)

	// Даем время на завершение горутин
	time.Sleep(100 * time.Millisecond)
}

func TestNoLeakedGoroutines(t *testing.T) {
	// Этот тест труднее реализовать напрямую без инструментов вроде runtime.NumGoroutine
	// Будем проверять косвенно через проверку закрытия каналов

	sp := NewSubPub()
	subject := "leak-test"

	// Создаем сигнальный канал, который будет закрыт, если горутина продолжает работу
	signal := make(chan struct{})

	// Счетчик для отслеживания обработанных сообщений
	var counter int
	var mu sync.Mutex

	// Подписываемся
	cb := func(msg interface{}) {
		mu.Lock()
		counter++
		mu.Unlock()

		// Если получено 5 сообщений, сигнализируем
		if counter == 5 {
			close(signal)
		}
	}

	sub, err := sp.Subscribe(subject, cb)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Публикуем сообщения
	for i := 0; i < 5; i++ {
		if err := sp.Publish(subject, i); err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}
	}

	// Ждем обработки всех сообщений
	select {
	case <-signal:
		// OK, все сообщения обработаны
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for messages to be processed")
	}

	// Отписываемся
	sub.Unsubscribe()

	// Публикуем еще одно сообщение
	if err := sp.Publish(subject, "after-unsubscribe"); err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Ждем немного
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что счетчик не увеличился
	mu.Lock()
	if counter > 5 {
		t.Errorf("Counter increased after unsubscribe: %d", counter)
	}
	mu.Unlock()

	// Закрываем систему
	ctx := context.Background()
	if err := sp.Close(ctx); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	// Ждем немного
	time.Sleep(100 * time.Millisecond)

	// Публикуем сообщение после закрытия
	err = sp.Publish(subject, "after-close")
	if err == nil {
		t.Error("Expected error when publishing after close, got nil")
	}
}

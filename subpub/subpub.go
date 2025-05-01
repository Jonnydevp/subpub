package subpub

import (
	"context"
	"errors"
	"sync"
)

// MessageHandler is a callback function that processes messages delivered to subscribers.
type MessageHandler func(msg interface{})

type Subscription interface {
	// Unsubscribe will remove interest in the current subject subscription for this subscriber.
	Unsubscribe()
}

type SubPub interface {
	// Subscribe creates an asynchronous queue subscriber on the given subject.
	Subscribe(subject string, cb MessageHandler) (Subscription, error)

	// Publish publishes the msg argument to the given subject.
	Publish(subject string, msg interface{}) error

	// Close will shutdown sub-pub system.
	// May be blocked by data delivery until the context is canceled.
	Close(ctx context.Context) error
}

// Структуры данных для реализации

// subscriber представляет подписчика
type subscriber struct {
	id       uint64
	handler  MessageHandler
	msgChan  chan interface{}
	done     chan struct{}
	subpubWg *sync.WaitGroup
	quit     bool
	mu       sync.RWMutex
}

// subscription представляет подписку
type subscription struct {
	subject    string
	subscriber *subscriber
	subpub     *subPubImpl
}

// Реализация интерфейса Subscription
func (s *subscription) Unsubscribe() {
	s.subpub.unsubscribe(s.subject, s.subscriber)
}

// subPubImpl основная реализация SubPub
type subPubImpl struct {
	mu          sync.RWMutex
	subjects    map[string]map[uint64]*subscriber
	nextSubID   uint64
	wg          sync.WaitGroup
	isClosing   bool
	closingChan chan struct{}
}

// NewSubPub создает новый экземпляр SubPub
func NewSubPub() SubPub {
	return &subPubImpl{
		subjects:    make(map[string]map[uint64]*subscriber),
		nextSubID:   1,
		closingChan: make(chan struct{}),
	}
}

// Subscribe создает асинхронного подписчика на указанную тему
func (sp *subPubImpl) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if cb == nil {
		return nil, errors.New("callback is required")
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Проверяем, не закрывается ли система
	if sp.isClosing {
		return nil, errors.New("subpub is closing")
	}

	// Создаем нового подписчика
	sub := &subscriber{
		id:       sp.nextSubID,
		handler:  cb,
		msgChan:  make(chan interface{}, 100), // Буфер для сообщений
		done:     make(chan struct{}),
		subpubWg: &sp.wg,
	}
	sp.nextSubID++

	// Добавляем подписчика к теме
	if _, exists := sp.subjects[subject]; !exists {
		sp.subjects[subject] = make(map[uint64]*subscriber)
	}
	sp.subjects[subject][sub.id] = sub

	// Запускаем горутину для обработки сообщений
	sp.wg.Add(1)
	go sub.processMessages()

	return &subscription{
		subject:    subject,
		subscriber: sub,
		subpub:     sp,
	}, nil
}

// Publish публикует сообщение в указанную тему
func (sp *subPubImpl) Publish(subject string, msg interface{}) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	// Проверяем, не закрывается ли система
	if sp.isClosing {
		return errors.New("subpub is closing")
	}

	// Проверяем существование темы
	subscribers, exists := sp.subjects[subject]
	if !exists || len(subscribers) == 0 {
		return nil // Нет подписчиков, просто игнорируем сообщение
	}

	// Отправляем сообщение всем подписчикам
	for _, sub := range subscribers {
		sub.mu.RLock()
		if !sub.quit {
			select {
			case sub.msgChan <- msg:
				// Успешно отправили сообщение
			default:
				// Если канал заполнен, добавляем сообщение в порядке очереди
				go func(sub *subscriber, message interface{}) {
					sub.msgChan <- message
				}(sub, msg)
			}
		}
		sub.mu.RUnlock()
	}

	return nil
}

// Close закрывает систему subpub
func (sp *subPubImpl) Close(ctx context.Context) error {
	// Отмечаем, что система закрывается
	sp.mu.Lock()
	if sp.isClosing {
		sp.mu.Unlock()
		return errors.New("subpub is already closing")
	}
	sp.isClosing = true

	// Сигнализируем всем подписчикам о завершении
	for _, subscribers := range sp.subjects {
		for _, sub := range subscribers {
			sub.mu.Lock()
			if !sub.quit {
				sub.quit = true
				close(sub.done)
			}
			sub.mu.Unlock()
		}
	}

	sp.mu.Unlock()
	close(sp.closingChan)

	// Создаем канал для сигнала о завершении всех горутин
	done := make(chan struct{})

	// Запускаем горутину для ожидания завершения всех подписчиков
	go func() {
		sp.wg.Wait()
		close(done)
	}()

	// Ожидаем либо завершения всех горутин, либо отмены контекста
	select {
	case <-done:
		// Все горутины завершились
		return nil
	case <-ctx.Done():
		// Контекст отменен, выходим немедленно
		return ctx.Err()
	}
}

// unsubscribe удаляет подписчика из темы
func (sp *subPubImpl) unsubscribe(subject string, sub *subscriber) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Проверяем, существует ли тема
	subscribers, exists := sp.subjects[subject]
	if !exists {
		return
	}

	// Проверяем, существует ли подписчик
	if _, ok := subscribers[sub.id]; !ok {
		return
	}

	// Отмечаем подписчика как завершенного
	sub.mu.Lock()
	if !sub.quit {
		sub.quit = true
		close(sub.done)
	}
	sub.mu.Unlock()

	// Удаляем подписчика из списка
	delete(subscribers, sub.id)

	// Если больше нет подписчиков, удаляем тему
	if len(subscribers) == 0 {
		delete(sp.subjects, subject)
	}
}

// processMessages обрабатывает сообщения для подписчика
func (s *subscriber) processMessages() {
	defer s.subpubWg.Done()

	for {
		select {
		case msg := <-s.msgChan:
			// Вызываем обработчик подписчика
			s.handler(msg)
		case <-s.done:
			// Подписка была отменена
			return
		}
	}
}

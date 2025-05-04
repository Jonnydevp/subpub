package main

import (
	"context"
	"fmt"
	"gopubsub/subpub"
	"sync"
	"time"
)

func main() {
	// Создаем новый экземпляр SubPub
	sp := subpub.NewSubPub()

	var wg sync.WaitGroup
	wg.Add(1)

	// Подписываемся на тему "news"
	sub1, err := sp.Subscribe("news", func(msg interface{}) {
		fmt.Printf("Подписчик 1 получил: %v\n", msg)
	})
	if err != nil {
		fmt.Printf("Ошибка подписки: %v\n", err)
		return
	}

	// Второй подписчик на ту же тему
	_, err = sp.Subscribe("news", func(msg interface{}) {
		fmt.Printf("Подписчик 2 получил: %v\n", msg)
		time.Sleep(300 * time.Millisecond) // Медленный подписчик
	})
	if err != nil {
		fmt.Printf("Ошибка подписки: %v\n", err)
		return
	}

	// Подписчик на другую тему
	_, err = sp.Subscribe("weather", func(msg interface{}) {
		fmt.Printf("Погода: %v\n", msg)
		wg.Done()
	})
	if err != nil {
		fmt.Printf("Ошибка подписки: %v\n", err)
		return
	}

	// Публикуем несколько сообщений
	fmt.Println("Публикуем сообщения...")
	sp.Publish("news", "ВК-лучшая компания")
	sp.Publish("news", "Хочу на стажку")
	sp.Publish("weather", "Сегодня солнечно")

	// Отписываемся от "news" для первого подписчика
	fmt.Println("Отписываем первого подписчика...")
	sub1.Unsubscribe()

	// Публикуем еще одно сообщение
	sp.Publish("news", "Важная новость 3")

	// Ждем, чтобы увидеть результаты
	wg.Wait()
	fmt.Println("Ожидание завершения обработки...")
	time.Sleep(500 * time.Millisecond)

	// Закрываем систему
	fmt.Println("Закрываем систему...")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := sp.Close(ctx); err != nil {
		fmt.Printf("Ошибка закрытия: %v\n", err)
		return
	}

	fmt.Println("Система успешно закрыта")
}

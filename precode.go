package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	ctx1, cancel := context.WithTimeout(ctx, 200*time.Microsecond)
	defer cancel()

	for i := 0; i < 100; i++ {
		select {
		case <-ctx1.Done():
			close(ch)
			return
		case ch <- int64(i):
			fn(int64(i))
		}
	}

	close(ch)
}

func Worker(in <-chan int64, out chan<- int64) {
	for v := range in {
		out <- v
	}
	close(out)
}

func main() {
	chIn := make(chan int64)
	// 3. Создание контекста
	//
	ctx := context.Background()

	// для проверки будем считать количество и сумму отправленных чисел
	var inputSum int64   // сумма сгенерированных чисел
	var inputCount int64 // количество сгенерированных чисел

	// генерируем числа, считая параллельно их количество и сумму
	go Generator(ctx, chIn, func(i int64) {
		inputSum += i
		inputCount++
	})
	const NumOut = 5 // количество обрабатывающих горутин и каналов
	// outs — слайс каналов, куда будут записываться числа из chIn
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		// создаём каналы и для каждого из них вызываем горутину Worker
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	// amounts — слайс, в который собирается статистика по горутинам
	amounts := make([]int64, NumOut)
	// chOut — канал, в который будут отправляться числа из горутин `outs[i]`
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup
	wg.Add(NumOut)

	// 4. Собираем числа из каналов outs
	//
	for i := 0; i < NumOut; i++ {
		go func(i int) {
			defer wg.Done()
			var sum int64
			var count int64
			for v := range outs[i] {
				sum += v
				count++
			}
			amounts[i] = count
			chOut <- sum
		}(i)
	}
	go func() {
		// ждём завершения работы всех горутин для outs
		wg.Wait()
		// закрываем результирующий канал
		close(chOut)
	}()

	var count1 int64 // количество чисел результирующего канала
	var sum1 int64   // сумма чисел результирующего канала

	// 5. Читаем числа из результирующего канала
	//
	for v := range chOut {
		sum1 += v
	}
	for _, v := range amounts {
		count1 += v
	}

	fmt.Println("Количество чисел", inputCount, count1)
	fmt.Println("Сумма чисел", inputSum, sum1)
	fmt.Println("Разбивка по каналам", amounts)

	// проверка результатов
	if inputSum != sum1 {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum1)
	}
	if inputCount != count1 {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count1)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}

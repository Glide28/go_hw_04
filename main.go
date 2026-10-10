package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	ledger := NewLedger()

	demoTransaction := Transaction{
		Amount:   750,
		Category: "Книги",
		Date:     time.Now(),
	}
	demoBudget := Budget{Category: "Книги", Limit: 3000}

	for name, item := range map[string]Validatable{
		"транзакция": demoTransaction,
		"бюджет":     demoBudget,
	} {
		if err := CheckValid(item); err != nil {
			log.Printf("Объект %q не прошёл проверку: %v", name, err)
			continue
		}
		fmt.Printf("Объект %q прошёл проверку Validatable\n", name)
	}

	file, err := os.Open("budgets.json")
	if err != nil {
		log.Fatalf("Не удалось открыть файл бюджетов: %v", err)
	}
	defer file.Close()

	if err := ledger.LoadBudgets(bufio.NewReader(file)); err != nil {
		log.Fatalf("Не удалось загрузить бюджеты: %v", err)
	}

	examples := []Transaction{
		{
			Amount:      1500,
			Category:    "Еда",
			Description: "Покупка продуктов",
			Date:        time.Now(),
		},
		{
			Amount:      2000,
			Category:    "Еда",
			Description: "Ужин в ресторане",
			Date:        time.Now(),
		},
		{
			Amount:      1800,
			Category:    "Еда",
			Description: "Эта операция превысит бюджет",
			Date:        time.Now(),
		},
		{
			Amount:      900,
			Category:    "Транспорт",
			Description: "Пополнение транспортной карты",
			Date:        time.Now(),
		},
	}

	for _, tx := range examples {
		if err := ledger.AddTransaction(tx); err != nil {
			if errors.Is(err, ErrBudgetExceeded) {
				fmt.Printf("Транзакция отклонена: %v\n", err)
				continue
			}
			log.Printf("Не удалось добавить транзакцию: %v", err)
		}
	}

	fmt.Println("\nСохранённые транзакции:")
	for _, tx := range ledger.ListTransactions() {
		fmt.Printf("%d. %s — %.2f руб. (%s)\n", tx.ID, tx.Category, tx.Amount, tx.Description)
	}
}

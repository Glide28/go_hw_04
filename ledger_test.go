package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func validTransaction(amount float64, category string) Transaction {
	return Transaction{Amount: amount, Category: category, Date: time.Now()}
}

func TestAddTransactionWithinBudget(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.SetBudget(Budget{Category: "Еда", Limit: 5000}); err != nil {
		t.Fatalf("не удалось установить бюджет: %v", err)
	}

	if err := ledger.AddTransaction(validTransaction(1200, "Еда")); err != nil {
		t.Fatalf("транзакция в пределах бюджета отклонена: %v", err)
	}

	items := ledger.ListTransactions()
	if len(items) != 1 {
		t.Fatalf("ожидалась 1 транзакция, получено %d", len(items))
	}
	if items[0].ID != 1 {
		t.Fatalf("ожидался ID 1, получен %d", items[0].ID)
	}
}

func TestAddTransactionRejectsBudgetExceeded(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.SetBudget(Budget{Category: "Еда", Limit: 1000}); err != nil {
		t.Fatalf("не удалось установить бюджет: %v", err)
	}
	if err := ledger.AddTransaction(validTransaction(700, "Еда")); err != nil {
		t.Fatalf("не удалось добавить первую транзакцию: %v", err)
	}

	err := ledger.AddTransaction(validTransaction(400, "Еда"))
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("ожидалась ошибка ErrBudgetExceeded, получено: %v", err)
	}
	if len(ledger.ListTransactions()) != 1 {
		t.Fatal("транзакция, превышающая бюджет, не должна сохраняться")
	}
}

func TestSetBudgetUpdatesLimit(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.SetBudget(Budget{Category: "Еда", Limit: 1000}); err != nil {
		t.Fatalf("не удалось установить бюджет: %v", err)
	}
	if err := ledger.SetBudget(Budget{Category: "Еда", Limit: 2000}); err != nil {
		t.Fatalf("не удалось обновить бюджет: %v", err)
	}

	if err := ledger.AddTransaction(validTransaction(1500, "Еда")); err != nil {
		t.Fatalf("обновлённый лимит не применился: %v", err)
	}
}

func TestLoadBudgets(t *testing.T) {
	ledger := NewLedger()
	data := `[
		{"category":"Еда","limit":5000,"period":"месяц"},
		{"category":"Транспорт","limit":3000,"period":"месяц"}
	]`

	if err := ledger.LoadBudgets(strings.NewReader(data)); err != nil {
		t.Fatalf("не удалось загрузить бюджеты: %v", err)
	}
	if len(ledger.budgets) != 2 {
		t.Fatalf("ожидалось 2 бюджета, получено %d", len(ledger.budgets))
	}
}

func TestLoadBudgetsReturnsReadableError(t *testing.T) {
	ledger := NewLedger()
	err := ledger.LoadBudgets(strings.NewReader(`[{"category":"Еда","limit":}]`))
	if err == nil {
		t.Fatal("ожидалась ошибка разбора JSON")
	}
	if !strings.Contains(err.Error(), "не удалось прочитать бюджеты из JSON") {
		t.Fatalf("ошибка должна объяснять причину: %v", err)
	}
}

func TestTransactionWithoutBudgetIsAllowed(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.AddTransaction(validTransaction(500, "Прочее")); err != nil {
		t.Fatalf("категория без бюджета должна быть разрешена: %v", err)
	}
}

func TestListTransactionsReturnsCopy(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.AddTransaction(validTransaction(100, "Прочее")); err != nil {
		t.Fatalf("не удалось добавить транзакцию: %v", err)
	}

	items := ledger.ListTransactions()
	items[0].Amount = 999
	if ledger.ListTransactions()[0].Amount != 100 {
		t.Fatal("ListTransactions должна возвращать копию хранилища")
	}
}

func TestTransactionValidate(t *testing.T) {
	tests := []struct {
		name string
		tx   Transaction
	}{
		{name: "неположительная сумма", tx: validTransaction(0, "Еда")},
		{name: "пустая категория", tx: validTransaction(100, "   ")},
		{name: "не задана дата", tx: Transaction{Amount: 100, Category: "Еда"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.tx.Validate(); err == nil {
				t.Fatal("ожидалась ошибка валидации")
			}
		})
	}
}

func TestBudgetValidate(t *testing.T) {
	if err := (Budget{Category: "Еда", Limit: 1000}).Validate(); err != nil {
		t.Fatalf("корректный бюджет не прошёл проверку: %v", err)
	}
	if err := (Budget{Category: "Еда", Limit: -1}).Validate(); err == nil {
		t.Fatal("бюджет с отрицательным лимитом должен быть отклонён")
	}
}

func TestCheckValidPolymorphism(t *testing.T) {
	items := []Validatable{
		validTransaction(100, "Еда"),
		Budget{Category: "Еда", Limit: 1000},
	}
	for _, item := range items {
		if err := CheckValid(item); err != nil {
			t.Fatalf("объект %T не прошёл проверку: %v", item, err)
		}
	}
}

func TestInvalidDataIsNotStored(t *testing.T) {
	ledger := NewLedger()
	if err := ledger.AddTransaction(Transaction{Amount: 100, Category: "Еда"}); err == nil {
		t.Fatal("транзакция без даты должна быть отклонена")
	}
	if len(ledger.ListTransactions()) != 0 {
		t.Fatal("невалидная транзакция не должна сохраняться")
	}
	if err := ledger.SetBudget(Budget{Category: "Еда", Limit: 0}); err == nil {
		t.Fatal("бюджет с нулевым лимитом должен быть отклонён")
	}
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ErrBudgetExceeded возвращается, когда новая транзакция превышает бюджет.
var ErrBudgetExceeded = errors.New("budget exceeded")

// Validatable задаёт единый контракт проверки данных.
type Validatable interface {
	Validate() error
}

// Transaction описывает финансовую операцию.
type Transaction struct {
	ID          int       `json:"id"`
	Amount      float64   `json:"amount"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
}

// Validate проверяет, что транзакция готова к сохранению.
func (tx Transaction) Validate() error {
	if tx.Amount <= 0 {
		return errors.New("сумма транзакции должна быть положительной")
	}
	if strings.TrimSpace(tx.Category) == "" {
		return errors.New("категория транзакции не может быть пустой")
	}
	if tx.Date.IsZero() {
		return errors.New("дата транзакции не задана")
	}
	return nil
}

// Budget содержит лимит расходов для категории за указанный период.
type Budget struct {
	Category string  `json:"category"`
	Limit    float64 `json:"limit"`
	Period   string  `json:"period,omitempty"`
}

// Validate проверяет данные бюджета.
func (b Budget) Validate() error {
	if b.Limit <= 0 {
		return errors.New("лимит бюджета должен быть положительным")
	}
	if strings.TrimSpace(b.Category) == "" {
		return errors.New("категория бюджета не может быть пустой")
	}
	return nil
}

// CheckValid демонстрирует полиморфную проверку объектов.
func CheckValid(v Validatable) error {
	if v == nil {
		return errors.New("объект для проверки не задан")
	}
	return v.Validate()
}

// Ledger хранит транзакции и бюджеты в памяти.
type Ledger struct {
	mu           sync.RWMutex
	transactions []Transaction
	budgets      map[string]Budget
	nextID       int
}

// NewLedger создаёт пустой сервис Ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions: make([]Transaction, 0),
		budgets:      make(map[string]Budget),
		nextID:       1,
	}
}

// SetBudget добавляет новый бюджет или обновляет существующий.
func (l *Ledger) SetBudget(b Budget) error {
	b.Category = strings.TrimSpace(b.Category)
	if err := CheckValid(b); err != nil {
		return fmt.Errorf("некорректный бюджет: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.budgets[b.Category] = b
	return nil
}

// AddTransaction проверяет бюджет и сохраняет транзакцию.
func (l *Ledger) AddTransaction(tx Transaction) error {
	tx.Category = strings.TrimSpace(tx.Category)
	if err := CheckValid(tx); err != nil {
		return fmt.Errorf("некорректная транзакция: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if budget, ok := l.budgets[tx.Category]; ok {
		spent := l.spentByCategoryLocked(tx.Category)
		if spent+tx.Amount > budget.Limit {
			return fmt.Errorf(
				"%w: категория %q, лимит %.2f руб., текущие расходы %.2f руб., новая операция %.2f руб.",
				ErrBudgetExceeded,
				tx.Category,
				budget.Limit,
				spent,
				tx.Amount,
			)
		}
	}

	tx.ID = l.nextID
	l.nextID++
	l.transactions = append(l.transactions, tx)
	return nil
}

// spentByCategoryLocked считает расходы по категории.
// Метод вызывается только при уже установленной блокировке l.mu.
func (l *Ledger) spentByCategoryLocked(category string) float64 {
	var total float64
	for _, tx := range l.transactions {
		if tx.Category == category {
			total += tx.Amount
		}
	}
	return total
}

// ListTransactions возвращает копию списка транзакций.
func (l *Ledger) ListTransactions() []Transaction {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]Transaction, len(l.transactions))
	copy(result, l.transactions)
	return result
}

// LoadBudgets читает массив бюджетов из JSON и добавляет их в Ledger.
func (l *Ledger) LoadBudgets(r io.Reader) error {
	if r == nil {
		return errors.New("источник бюджетов не задан")
	}

	var budgets []Budget
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&budgets); err != nil {
		return fmt.Errorf("не удалось прочитать бюджеты из JSON: %w", err)
	}
	if len(budgets) == 0 {
		return errors.New("файл бюджетов не содержит записей")
	}

	for i, budget := range budgets {
		if err := l.SetBudget(budget); err != nil {
			return fmt.Errorf("некорректный бюджет с индексом %d: %w", i, err)
		}
	}
	return nil
}

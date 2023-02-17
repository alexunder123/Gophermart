package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog/log"

	"gophermart/internal/config"
)

type SQLStorage struct {
	DB *sql.DB
}

func NewSQLStorager(cfg *config.Config) *SQLStorage {
	db, err := sql.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		log.Fatal().Err(err).Msg("Open DB sql error")
	}
	err = createDB(db)
	if err != nil {
		log.Fatal().Err(err).Msg("CreateDB create table error")
	}
	return &SQLStorage{
		DB: db,
	}
}

func createDB(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS gophermart_users(user_id text UNIQUE, login text UNIQUE, password text, balance integer DEFAULT 0, withdrawn integer DEFAULT 0;")
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gophermart_orders(order_no text UNIQUE, user_id text, status text DEFAULT NEW, accrual integer DEFAULT 0, date timestamptz DEFAULT current_timestamp;")
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gophermart_withdraws(order_no text UNIQUE, user_id text, sum integer, date timestamptz DEFAULT current_timestamp;")
	if err != nil {
		return err
	}
	// _, err = db.Exec("CREATE TABLE IF NOT EXISTS gophermart_temp(order_no text UNIQUE, status text DEFAULT new, date timestamptz DEFAULT current_timestamp;")
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (s *SQLStorage) CloseDB() {
	err := s.DB.Close()
	if err != nil {
		log.Error().Err(err).Msg("CloseDB DB closing err")
	}
	log.Info().Msg("db closed")
}

func (s *SQLStorage) AddNewUser(login, password, userID string) error {
	result, err := s.DB.Exec("INSERT INTO gophermart_users(user_id, login, password) VALUES($1, $2, $3) ON CONFLICT DO NOTHING", userID, login, password)
	if err != nil {
		return err
	}
	changes, _ := result.RowsAffected()
	if changes == 0 {
		return ErrConflict
	}
	return nil
}

func (s *SQLStorage) LogInUser(login, password string) (string, error) {
	var userID string
	row := s.DB.QueryRow("SELECT user_id FROM gophermart_users WHERE login = $1 AND password = $2", login, password)
	if errors.Is(row.Err(), sql.ErrNoRows) {
		return "", ErrAuthError
	}
	if row.Err() != nil {
		return "", row.Err()
	}
	err := row.Scan(&userID)
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *SQLStorage) CheckUser(userID string) error {
	row := s.DB.QueryRow("SELECT login FROM gophermart_users WHERE user_id = $1", userID)
	if errors.Is(row.Err(), sql.ErrNoRows) {
		return ErrAuthError
	}
	if row.Err() != nil {
		return row.Err()
	}
	return nil
}

func (s *SQLStorage) AddNewOrder(userID, orders string) error {
	var currenUser string
	row := s.DB.QueryRow("SELECT user_id FROM gophermart_users WHERE orders = $1", orders)
	if !errors.Is(row.Err(), sql.ErrNoRows) {
		err := row.Scan(&currenUser)
		if err != nil {
			return err
		}
		if userID == currenUser {
			return ErrUploaded
		}
		return ErrAnotherUserUploaded
	}

	_, err := s.DB.Exec("INSERT INTO gophermart_orders(order_no, user_id) VALUES($1, $2)", orders, userID)
	if err != nil {
		return err
	}
	// _, err = s.DB.Exec("INSERT INTO gophermart_temp(order_no) VALUES($1)", orders)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (s *SQLStorage) UserWithdraw(userID, order string, sum float64) error {
	var balance int
	row := s.DB.QueryRow("SELECT balance FROM gophermart_users WHERE user_id = $1", userID)
	if row.Err() != nil {
		return row.Err()
	}
	err := row.Scan(&balance)
	if err != nil {
		return err
	}
	if balance < int(sum*100) {
		return ErrNotEnouthBalance
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmtUser, err := tx.Prepare("UPDATE gophermart_users SET balance=$1 WHERE user_id = $2")
	if err != nil {
		return err
	}
	defer stmtUser.Close()
	stmtWdraw, err := tx.Prepare("INSERT INTO gophermart_withdraws(order_no, user_id, sum) VALUES($1, $2, $3)")
	if err != nil {
		return err
	}
	defer stmtWdraw.Close()
	if _, err = stmtUser.Exec(balance-int(sum*100), userID); err != nil {
		if err = tx.Rollback(); err != nil {
			log.Fatal().Msgf("update drivers: unable to rollback: %v", err)
		}
		return err
	}
	if _, err = stmtWdraw.Exec(order, userID, sum*100); err != nil {
		if err = tx.Rollback(); err != nil {
			log.Fatal().Msgf("update drivers: unable to rollback: %v", err)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Fatal().Msgf("update drivers: unable to commit: %v", err)
		return err
	}
	return nil
}

func (s *SQLStorage) UserBalance(userID string) ([]byte, error) {
	var balance, withdrawn int
	row := s.DB.QueryRow("SELECT balance, withdrawn FROM gophermart_users WHERE user_id = $1", userID)
	if row.Err() != nil {
		return nil, row.Err()
	}
	err := row.Scan(&balance, &withdrawn)
	if err != nil {
		return nil, err
	}
	currentUserBalance := currentBalance{Current: float64(balance) / 100, Withdrawn: float64(withdrawn) / 100}
	currentUserBalanceBZ, err := json.Marshal(currentUserBalance)
	if err != nil {
		return nil, err
	}
	return currentUserBalanceBZ, nil
}

func (s *SQLStorage) UserOrders(userID string) ([]byte, error) {
	var order_no, status string
	var accrual int
	var date time.Time
	currentUserOrders := make([]orders, 0)
	rows, err := s.DB.Query("SELECT order_no, status, accrual, date FROM gophermart_orders WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		i := 0
		err = rows.Scan(&order_no, &status, &accrual, &date)
		if err != nil {
			return nil, err
		}
		currentUserOrders = append(currentUserOrders, orders{Number: order_no, Status: status, UploadedAt: date.Format(time.RFC3339)})
		if status == "PROCESSED" {
			currentUserOrders[i].Accrual = float64(accrual) / 100
		}
		i++
	}
	currentUserOrdersBZ, err := json.Marshal(currentUserOrders)
	if err != nil {
		return nil, err
	}
	return currentUserOrdersBZ, nil
}

func (s *SQLStorage) UserWithdrawals(userID string) ([]byte, error) {
	var order_no string
	var sum int
	var date time.Time
	currentUserWithdraws := make([]withdraws, 0)
	rows, err := s.DB.Query("SELECT order_no, sum, date FROM gophermart_withdraws WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&order_no, &sum, &date)
		if err != nil {
			return nil, err
		}
		currentUserWithdraws = append(currentUserWithdraws, withdraws{Order: order_no, Sum: float64(sum) / 100, ProcessedAt: date.Format(time.RFC3339)})
	}
	currentUserWithdrawsBZ, err := json.Marshal(currentUserWithdraws)
	if err != nil {
		return nil, err
	}
	return currentUserWithdrawsBZ, nil
}

func (s *SQLStorage) GetProcessedOrders() ([]ProcessedOrders, error) {
	var order_no string
	var status string
	orders := make([]ProcessedOrders, 0)
	rows, err := s.DB.Query("SELECT order_no, status FROM gophermart_orders WHERE status=NEW OR status=REGISTERED OR status=PROCESSING ORDER BY date")
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&order_no, &status)
		if err != nil {
			return nil, err
		}
		orders = append(orders, ProcessedOrders{Order: order_no, Status: status})
	}
	return orders, nil
}

func (s *SQLStorage) UpdateOrderStatus(accResult AccuralResult) error {
	if accResult.Status == "PROCESSED"{
		_, err := s.DB.Exec("INSERT INTO gophermart_orders(status, accrual) VALUES(PROCESSED, $1) WHERE order_no=$2", int(accResult.Accrual*100), accResult.Order)
		if err != nil {
			return err
		}
		return nil
	}
	_, err := s.DB.Exec("INSERT INTO gophermart_orders(status) VALUES($1) WHERE order_no=$2", accResult.Status, accResult.Order)
		if err != nil {
			return err
		}
		return nil
	
}

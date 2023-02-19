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

	_, err := db.Exec("DROP TABLE IF EXISTS gophermart_orders;")
	if err != nil {
		log.Fatal().Err(err).Msg("CreateDB drop table error")
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gophermart_users(user_id text UNIQUE, login text UNIQUE, password text, balance integer DEFAULT 0, withdrawn integer DEFAULT 0);")
	if err != nil {
		return err
	}
	log.Debug().Msg("storage gophermart_users init")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gophermart_orders(order_no text UNIQUE, user_id text, status text DEFAULT 'NEW', accrual integer DEFAULT 0, date text);")
	if err != nil {
		return err
	}
	log.Debug().Msg("storage gophermart_orders init")
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS gophermart_withdraws(order_no text UNIQUE, user_id text, sum integer, date text);")
	if err != nil {
		return err
	}
	log.Debug().Msg("storage gophermart_withdraws init")
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
	err := s.DB.QueryRow("SELECT user_id FROM gophermart_users WHERE login = $1 AND password = $2", login, password).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrAuthError
	}
	if err != nil {
		return "", err
	}
	return userID, nil
}

func (s *SQLStorage) CheckUser(userID string) error {
	var login string
	err := s.DB.QueryRow("SELECT login FROM gophermart_users WHERE user_id = $1", userID).Scan(&login)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrAuthError
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *SQLStorage) AddNewOrder(userID, order string) error {
	var currenUser string
	err := s.DB.QueryRow("SELECT user_id FROM gophermart_orders WHERE order_no = $1", order).Scan(&currenUser)
	if !errors.Is(err, sql.ErrNoRows) {
		log.Debug().Msg("AddNewOrder order_id is present in DB")
		if userID == currenUser {
			return ErrUploaded
		}
		return ErrAnotherUserUploaded
	}
	today := time.Now()
	_, err = s.DB.Exec("INSERT INTO gophermart_orders(order_no, user_id, date) VALUES($1, $2, $3)", order, userID, today.Format(time.RFC3339))
	if err != nil {
		return err
	}
	// _, err = s.DB.Exec("INSERT INTO gophermart_temp(order_no) VALUES($1)", orders)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (s *SQLStorage) UserWithdraw(userID, order string, sum float32) error {
	var balance, withdrawn int
	err := s.DB.QueryRow("SELECT balance, withdrawn FROM gophermart_users WHERE user_id = $1", userID).Scan(&balance, &withdrawn)
	if err != nil {
		return err
	}
	log.Debug().Msgf("UserBalance: %d, %d, want to witdraw: %f", balance, withdrawn, sum)
	if balance < int(sum*100) {
		return ErrNotEnouthBalance
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmtUser, err := tx.Prepare("UPDATE gophermart_users SET balance=$1, withdrawn=$2 WHERE user_id = $3")
	if err != nil {
		return err
	}
	defer stmtUser.Close()
	stmtWdraw, err := tx.Prepare("INSERT INTO gophermart_withdraws(order_no, user_id, sum, date) VALUES($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	defer stmtWdraw.Close()
	today := time.Now()
	log.Debug().Msgf("newUserBalance: %d", balance-int(sum*100))
	if _, err = stmtUser.Exec(balance-int(sum*100), withdrawn+int(sum*100), userID); err != nil {
		if err = tx.Rollback(); err != nil {
			log.Fatal().Msgf("update drivers: unable to rollback: %v", err)
		}
		return err
	}
	if _, err = stmtWdraw.Exec(order, userID, int(sum*100), today.Format(time.RFC3339)); err != nil {
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
	err := s.DB.QueryRow("SELECT balance, withdrawn FROM gophermart_users WHERE user_id = $1", userID).Scan(&balance, &withdrawn)
	if err != nil {
		return nil, err
	}
	currentUserBalance := currentBalance{Current: float32(balance) / 100, Withdrawn: float32(withdrawn) / 100}
	log.Debug().Msgf("currentUserBalance: %f", currentUserBalance)
	currentUserBalanceBZ, err := json.Marshal(currentUserBalance)
	if err != nil {
		return nil, err
	}
	return currentUserBalanceBZ, nil
}

func (s *SQLStorage) UserOrders(userID string) ([]byte, error) {
	var orderNo, status, date string
	var accrual int
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
		err = rows.Scan(&orderNo, &status, &accrual, &date)
		if err != nil {
			return nil, err
		}
		currentUserOrders = append(currentUserOrders, orders{Number: orderNo, Status: status, UploadedAt: date})
		if status == "PROCESSED" {
			currentUserOrders[i].Accrual = float32(accrual) / 100
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
	var orderNo, date string
	var sum int
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
		err = rows.Scan(&orderNo, &sum, &date)
		if err != nil {
			return nil, err
		}
		currentUserWithdraws = append(currentUserWithdraws, withdraws{Order: orderNo, Sum: float32(sum) / 100, ProcessedAt: date})
	}
	currentUserWithdrawsBZ, err := json.Marshal(currentUserWithdraws)
	if err != nil {
		return nil, err
	}
	return currentUserWithdrawsBZ, nil
}

func (s *SQLStorage) GetProcessedOrders() ([]ProcessedOrders, error) {
	var userID, orderNo, status string
	orders := make([]ProcessedOrders, 0)
	rows, err := s.DB.Query("SELECT user_id, order_no, status FROM gophermart_orders WHERE status='NEW' OR status='REGISTERED' OR status='PROCESSING' ORDER BY date")
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&userID, &orderNo, &status)
		if err != nil {
			return nil, err
		}
		orders = append(orders, ProcessedOrders{UserID: userID, Order: orderNo, Status: status})
	}
	return orders, nil
}

func (s *SQLStorage) UpdateOrderStatus(accResult AccuralResult) error {
	if accResult.Status == "PROCESSED" {
		var balance int
		err := s.DB.QueryRow("SELECT balance FROM gophermart_users WHERE user_id = $1", accResult.UserID).Scan(&balance)
		if err != nil {
			return err
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
		stmtOrder, err := tx.Prepare("UPDATE gophermart_orders SET status='PROCESSED', accrual=$1 WHERE order_no=$2")
		if err != nil {
			return err
		}
		defer stmtOrder.Close()

		if _, err = stmtUser.Exec(balance+int(accResult.Accrual*100), accResult.UserID); err != nil {
			if err = tx.Rollback(); err != nil {
				log.Fatal().Msgf("update drivers: unable to rollback: %v", err)
			}
			return err
		}
		if _, err = stmtOrder.Exec(int(accResult.Accrual*100), accResult.Order); err != nil {
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

	_, err := s.DB.Exec("UPDATE gophermart_orders SET status=$1 WHERE order_no=$2", accResult.Status, accResult.Order)
	if err != nil {
		return err
	}
	return nil

}

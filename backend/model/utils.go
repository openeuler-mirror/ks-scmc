package model

import (
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var ErrDuplicateKey = errors.New("duplicate key conflict")
var ErrRecordNotFound = gorm.ErrRecordNotFound

func translateError(err error) error {
	if isDuplicateKeyError(err) {
		return ErrDuplicateKey
	}

	return err
}

func isDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

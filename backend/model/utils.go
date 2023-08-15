package model

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

var ErrDBRecordNotFound = errors.New("database record not found")
var ErrDuplicateKey = errors.New("duplicate key conflict")

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

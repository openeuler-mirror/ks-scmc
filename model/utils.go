package model

import (
	"errors"
	"strings"

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

func MysqlEscape(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		escape := byte(0)

		switch c {
		case 0:
			escape = '0'
			break
		case '\n':
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '"':
			escape = '"'
			break
		case '\032':
			escape = 'Z'
		}

		if escape != 0 {
			sb.WriteByte('\\')
			sb.WriteByte(escape)
		} else {
			sb.WriteByte(c)
		}
	}

	return sb.String()
}

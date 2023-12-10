package errorsstorage

import "errors"

var ErrLoginCOnflict = errors.New("login alredy exist")
var ErrUserNotExists = errors.New("the login/password pair does not exist")
var ErrOrderNotExist = errors.New("order does not exist")
var ErrOrdersNotExist = errors.New("orders does not exists, list is empty")
var ErrWriteOffNotExist = errors.New("write off does not exists, list is empty")
var ErrDataBaseNoChange = errors.New("data base has not change, migration is not complete")

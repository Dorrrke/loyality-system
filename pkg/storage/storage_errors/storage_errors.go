package storage_errors

import "errors"

var ErrLoginCOnflict = errors.New("login alredy exist")
var ErrUserNotExists = errors.New("the login/password pair does not exist")
var ErrOrderNotExist = errors.New("order does not exist")
var ErrOrdersNotExist = errors.New("orders does not exists, list is empty")

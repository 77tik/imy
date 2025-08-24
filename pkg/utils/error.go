package utils

type BaseError struct {
	Code    int
	Message string
	Err     error
}

func (e *BaseError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *BaseError) WithError(err error) *BaseError {
	return &BaseError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

func (e *BaseError) GetCode() int {
	return e.Code
}

func (e *BaseError) GetMsg() string {
	return e.Message
}

func (e *BaseError) GetErr() error {
	return e.Err
}

func NewBaseError(code int, msg string) *BaseError {
	return &BaseError{
		Code:    code,
		Message: msg,
	}
}

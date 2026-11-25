package domain

// ErrorKind 决定 HTTP 状态码：bad_request(400) / unprocessable(422)。
type ErrorKind string

const (
	ErrBadRequest    ErrorKind = "bad_request"
	ErrUnprocessable ErrorKind = "unprocessable"
)

// AppError 携带可展示的诊断信息。
type AppError struct {
	Kind        ErrorKind
	Message     string
	Diagnostics []Diagnostic
}

func (e *AppError) Error() string { return e.Message }

func NewBadRequest(msg string, diags ...Diagnostic) *AppError {
	return &AppError{Kind: ErrBadRequest, Message: msg, Diagnostics: diags}
}

func NewUnprocessable(msg string, diags ...Diagnostic) *AppError {
	return &AppError{Kind: ErrUnprocessable, Message: msg, Diagnostics: diags}
}

package apperr

type AppError struct {
	Code    string
	Message string
	Status  int
	Fields  map[string][]string
	Err     error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:    "NOT_FOUND",
		Message: resource + " not found",
		Status:  404,
	}
}

func NewValidationError(message string, fields map[string][]string) *AppError {
	return &AppError{
		Code:    "VALIDATION_ERROR",
		Message: message,
		Status:  400,
		Fields:  fields,
	}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  401,
	}
}

func NewConflictError(message string) *AppError {
	return &AppError{
		Code:    "CONFLICT",
		Message: message,
		Status:  409,
	}
}

func NewInternalError(err error) *AppError {
	return &AppError{
		Code:    "INTERNAL",
		Message: "internal server error",
		Status:  500,
		Err:     err,
	}
}

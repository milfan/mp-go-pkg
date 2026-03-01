// Package pkgerrors is for wrapping error
package pkgerrors

var errorDicts *ErrDicts = &ErrDicts{
	Errors: make(map[string]*Error),
}

type ErrDicts struct {
	Errors map[string]*Error
}

func RegisterDicts(errCodes map[string]*Error) {
	errorDicts = &ErrDicts{
		Errors: errCodes,
	}
}

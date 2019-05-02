package main

import (
	"io"
	"log"
	"net/http"

	"golang.org/x/xerrors"
)

type statusError struct {
	Code int
	Msg  string
	Err  error
}

func (e *statusError) Error() string {
	return e.Msg
}

func (e *statusError) Unwrap() error {
	return e.Err
}

func StatusError(code int, msg string, cause error) *statusError {
	return &statusError{code, msg, cause}
}

func HandleErrorHTTP(err error, w http.ResponseWriter, r *http.Request) {
	if err == nil {
		return
	}

	var (
		statusErr  *statusError
		statusCode int
		errMsg     string
	)
	if xerrors.As(err, &statusErr) {
		statusCode = statusErr.Code
		errMsg = statusErr.Error()
	} else {
		statusCode = http.StatusInternalServerError
		errMsg = "internal error"
	}

	w.WriteHeader(statusCode)

	io.WriteString(w, errMsg)

	if origErr := xerrors.Unwrap(err); origErr != nil {
		err = origErr
	}
	// unwrapped error can be nil, so double check
	if err != nil {
		log.Printf("request failed for url %s: %v", r.URL.String(), err)
	}
}

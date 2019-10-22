package errormessages

import (
	"encoding/json"
	"net/http"
)

// ErrorMessage represents a typical error response
type ErrorMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ErrorMessageInterface a generic json error message
type ErrorMessageInterface struct {
	Status  string      `json:"status"`
	Message interface{} `json:"messages"`
}

// WriteErrorInterface shows an error with a interface as json message
func WriteErrorInterface(w http.ResponseWriter, message interface{},
	status int) {
	errResponse := &ErrorMessageInterface{
		Status:  "ERROR",
		Message: message,
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errResponse)
}

// WriteErrorMessage writes the error the response if there is any
func WriteErrorMessage(w http.ResponseWriter, message string, status int) {
	errResponse := &ErrorMessage{
		Status:  "ERROR",
		Message: message,
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errResponse)
}

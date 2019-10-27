package pagination

import (
	"errors"
)

// Page is a struct to represente a resourse pagination
type Page struct {
	Data      interface{} `json:"data"`
	Size      int         `json:"size"`
	NextToken int64       `json:"nextToken"`
}

// Validate validates pagination characteristics
func (p *Page) Validate() error {
	if p.Size <= 0 {
		return errors.New("Invalid page size")
	}

	if p.Size > 200 {
		return errors.New("Page limit size exceeded")
	}

	if p.NextToken < 0 {
		return errors.New("Invalid next token")
	}
	return nil
}

package db

import "fmt"

type MockDB struct {
	isError bool
}

func NewMockDB(isError bool) *MockDB { return &MockDB{isError} }

func (m *MockDB) Store(object interface{}) error {
	if m.isError {
		return fmt.Errorf("error")
	}

	return nil
}

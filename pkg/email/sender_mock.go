package email

import "github.com/stretchr/testify/mock"

var _ ISender = &MockSender{}

type MockSender struct {
	mock.Mock
}

// SendEmail : Sends a plain text email to the client
func (m *MockSender) SendEmail(c *Content) error {
	a := m.Called(c)
	return a.Error(0)
}

// SendHTMLEmail : Sends an email that is formatted with HTML
func (m *MockSender) SendHTMLEmail(c *Content) error {
	return m.SendEmail(c)
}

func NewMockSender() *MockSender {
	return new(MockSender)
}

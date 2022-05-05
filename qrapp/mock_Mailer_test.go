// Code generated by mockery. DO NOT EDIT.

package qrapp

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockMailer is an autogenerated mock type for the Mailer type
type MockMailer struct {
	mock.Mock
}

// SendReply provides a mock function with given fields: ctx, messageID, from, to, subject, text, html
func (_m *MockMailer) SendReply(ctx context.Context, messageID string, from string, to string, subject string, text string, html string) error {
	ret := _m.Called(ctx, messageID, from, to, subject, text, html)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string, string, string) error); ok {
		r0 = rf(ctx, messageID, from, to, subject, text, html)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/spacemeshos/poet/registration (interfaces: PowVerifier)
//
// Generated by this command:
//
//	mockgen -typed -package mocks -destination mocks/pow_verifier.go . PowVerifier
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	registration "github.com/spacemeshos/poet/registration"
	gomock "go.uber.org/mock/gomock"
)

// MockPowVerifier is a mock of PowVerifier interface.
type MockPowVerifier struct {
	ctrl     *gomock.Controller
	recorder *MockPowVerifierMockRecorder
}

// MockPowVerifierMockRecorder is the mock recorder for MockPowVerifier.
type MockPowVerifierMockRecorder struct {
	mock *MockPowVerifier
}

// NewMockPowVerifier creates a new mock instance.
func NewMockPowVerifier(ctrl *gomock.Controller) *MockPowVerifier {
	mock := &MockPowVerifier{ctrl: ctrl}
	mock.recorder = &MockPowVerifierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPowVerifier) EXPECT() *MockPowVerifierMockRecorder {
	return m.recorder
}

// Params mocks base method.
func (m *MockPowVerifier) Params() registration.PowParams {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Params")
	ret0, _ := ret[0].(registration.PowParams)
	return ret0
}

// Params indicates an expected call of Params.
func (mr *MockPowVerifierMockRecorder) Params() *MockPowVerifierParamsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Params", reflect.TypeOf((*MockPowVerifier)(nil).Params))
	return &MockPowVerifierParamsCall{Call: call}
}

// MockPowVerifierParamsCall wrap *gomock.Call
type MockPowVerifierParamsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockPowVerifierParamsCall) Return(arg0 registration.PowParams) *MockPowVerifierParamsCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockPowVerifierParamsCall) Do(f func() registration.PowParams) *MockPowVerifierParamsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockPowVerifierParamsCall) DoAndReturn(f func() registration.PowParams) *MockPowVerifierParamsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SetParams mocks base method.
func (m *MockPowVerifier) SetParams(arg0 registration.PowParams) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetParams", arg0)
}

// SetParams indicates an expected call of SetParams.
func (mr *MockPowVerifierMockRecorder) SetParams(arg0 any) *MockPowVerifierSetParamsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetParams", reflect.TypeOf((*MockPowVerifier)(nil).SetParams), arg0)
	return &MockPowVerifierSetParamsCall{Call: call}
}

// MockPowVerifierSetParamsCall wrap *gomock.Call
type MockPowVerifierSetParamsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockPowVerifierSetParamsCall) Return() *MockPowVerifierSetParamsCall {
	c.Call = c.Call.Return()
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockPowVerifierSetParamsCall) Do(f func(registration.PowParams)) *MockPowVerifierSetParamsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockPowVerifierSetParamsCall) DoAndReturn(f func(registration.PowParams)) *MockPowVerifierSetParamsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Verify mocks base method.
func (m *MockPowVerifier) Verify(arg0, arg1 []byte, arg2 uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Verify", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// Verify indicates an expected call of Verify.
func (mr *MockPowVerifierMockRecorder) Verify(arg0, arg1, arg2 any) *MockPowVerifierVerifyCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Verify", reflect.TypeOf((*MockPowVerifier)(nil).Verify), arg0, arg1, arg2)
	return &MockPowVerifierVerifyCall{Call: call}
}

// MockPowVerifierVerifyCall wrap *gomock.Call
type MockPowVerifierVerifyCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockPowVerifierVerifyCall) Return(arg0 error) *MockPowVerifierVerifyCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockPowVerifierVerifyCall) Do(f func([]byte, []byte, uint64) error) *MockPowVerifierVerifyCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockPowVerifierVerifyCall) DoAndReturn(f func([]byte, []byte, uint64) error) *MockPowVerifierVerifyCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

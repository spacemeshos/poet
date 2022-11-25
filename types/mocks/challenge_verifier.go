// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/spacemeshos/poet/types (interfaces: ChallengeVerifier)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/spacemeshos/poet/types"
)

// MockChallengeVerifier is a mock of ChallengeVerifier interface.
type MockChallengeVerifier struct {
	ctrl     *gomock.Controller
	recorder *MockChallengeVerifierMockRecorder
}

// MockChallengeVerifierMockRecorder is the mock recorder for MockChallengeVerifier.
type MockChallengeVerifierMockRecorder struct {
	mock *MockChallengeVerifier
}

// NewMockChallengeVerifier creates a new mock instance.
func NewMockChallengeVerifier(ctrl *gomock.Controller) *MockChallengeVerifier {
	mock := &MockChallengeVerifier{ctrl: ctrl}
	mock.recorder = &MockChallengeVerifierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChallengeVerifier) EXPECT() *MockChallengeVerifierMockRecorder {
	return m.recorder
}

// Verify mocks base method.
func (m *MockChallengeVerifier) Verify(arg0 context.Context, arg1, arg2 []byte) (*types.ChallengeVerificationResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Verify", arg0, arg1, arg2)
	ret0, _ := ret[0].(*types.ChallengeVerificationResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Verify indicates an expected call of Verify.
func (mr *MockChallengeVerifierMockRecorder) Verify(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Verify", reflect.TypeOf((*MockChallengeVerifier)(nil).Verify), arg0, arg1, arg2)
}
// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/spacemeshos/poet/service (interfaces: RegistrationService)
//
// Generated by this command:
//
//	mockgen -package mocks -destination mocks/service.go . RegistrationService
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	service "github.com/spacemeshos/poet/service"
	shared "github.com/spacemeshos/poet/shared"
	gomock "go.uber.org/mock/gomock"
)

// MockRegistrationService is a mock of RegistrationService interface.
type MockRegistrationService struct {
	ctrl     *gomock.Controller
	recorder *MockRegistrationServiceMockRecorder
	isgomock struct{}
}

// MockRegistrationServiceMockRecorder is the mock recorder for MockRegistrationService.
type MockRegistrationServiceMockRecorder struct {
	mock *MockRegistrationService
}

// NewMockRegistrationService creates a new mock instance.
func NewMockRegistrationService(ctrl *gomock.Controller) *MockRegistrationService {
	mock := &MockRegistrationService{ctrl: ctrl}
	mock.recorder = &MockRegistrationServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRegistrationService) EXPECT() *MockRegistrationServiceMockRecorder {
	return m.recorder
}

// NewProof mocks base method.
func (m *MockRegistrationService) NewProof(ctx context.Context, proof shared.NIP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewProof", ctx, proof)
	ret0, _ := ret[0].(error)
	return ret0
}

// NewProof indicates an expected call of NewProof.
func (mr *MockRegistrationServiceMockRecorder) NewProof(ctx, proof any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewProof", reflect.TypeOf((*MockRegistrationService)(nil).NewProof), ctx, proof)
}

// RegisterForRoundClosed mocks base method.
func (m *MockRegistrationService) RegisterForRoundClosed(ctx context.Context) <-chan service.ClosedRound {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RegisterForRoundClosed", ctx)
	ret0, _ := ret[0].(<-chan service.ClosedRound)
	return ret0
}

// RegisterForRoundClosed indicates an expected call of RegisterForRoundClosed.
func (mr *MockRegistrationServiceMockRecorder) RegisterForRoundClosed(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterForRoundClosed", reflect.TypeOf((*MockRegistrationService)(nil).RegisterForRoundClosed), ctx)
}

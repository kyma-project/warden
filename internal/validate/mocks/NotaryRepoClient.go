// Code generated by mockery v2.16.0. DO NOT EDIT.

package mocks

import (
	client "github.com/theupdateframework/notary/client"
	changelist "github.com/theupdateframework/notary/client/changelist"

	data "github.com/theupdateframework/notary/tuf/data"

	mock "github.com/stretchr/testify/mock"

	signed "github.com/theupdateframework/notary/tuf/signed"
)

// NotaryRepoClient is an autogenerated mock type for the NotaryRepoClient type
type NotaryRepoClient struct {
	mock.Mock
}

// AddDelegation provides a mock function with given fields: name, delegationKeys, paths
func (_m *NotaryRepoClient) AddDelegation(name data.RoleName, delegationKeys []data.PublicKey, paths []string) error {
	ret := _m.Called(name, delegationKeys, paths)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, []data.PublicKey, []string) error); ok {
		r0 = rf(name, delegationKeys, paths)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddDelegationPaths provides a mock function with given fields: name, paths
func (_m *NotaryRepoClient) AddDelegationPaths(name data.RoleName, paths []string) error {
	ret := _m.Called(name, paths)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, []string) error); ok {
		r0 = rf(name, paths)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddDelegationRoleAndKeys provides a mock function with given fields: name, delegationKeys
func (_m *NotaryRepoClient) AddDelegationRoleAndKeys(name data.RoleName, delegationKeys []data.PublicKey) error {
	ret := _m.Called(name, delegationKeys)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, []data.PublicKey) error); ok {
		r0 = rf(name, delegationKeys)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AddTarget provides a mock function with given fields: target, roles
func (_m *NotaryRepoClient) AddTarget(target *client.Target, roles ...data.RoleName) error {
	_va := make([]interface{}, len(roles))
	for _i := range roles {
		_va[_i] = roles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, target)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(*client.Target, ...data.RoleName) error); ok {
		r0 = rf(target, roles...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ClearDelegationPaths provides a mock function with given fields: name
func (_m *NotaryRepoClient) ClearDelegationPaths(name data.RoleName) error {
	ret := _m.Called(name)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAllTargetMetadataByName provides a mock function with given fields: name
func (_m *NotaryRepoClient) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	ret := _m.Called(name)

	var r0 []client.TargetSignedStruct
	if rf, ok := ret.Get(0).(func(string) []client.TargetSignedStruct); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]client.TargetSignedStruct)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetChangelist provides a mock function with given fields:
func (_m *NotaryRepoClient) GetChangelist() (changelist.Changelist, error) {
	ret := _m.Called()

	var r0 changelist.Changelist
	if rf, ok := ret.Get(0).(func() changelist.Changelist); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(changelist.Changelist)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCryptoService provides a mock function with given fields:
func (_m *NotaryRepoClient) GetCryptoService() signed.CryptoService {
	ret := _m.Called()

	var r0 signed.CryptoService
	if rf, ok := ret.Get(0).(func() signed.CryptoService); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(signed.CryptoService)
		}
	}

	return r0
}

// GetDelegationRoles provides a mock function with given fields:
func (_m *NotaryRepoClient) GetDelegationRoles() ([]data.Role, error) {
	ret := _m.Called()

	var r0 []data.Role
	if rf, ok := ret.Get(0).(func() []data.Role); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]data.Role)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetGUN provides a mock function with given fields:
func (_m *NotaryRepoClient) GetGUN() data.GUN {
	ret := _m.Called()

	var r0 data.GUN
	if rf, ok := ret.Get(0).(func() data.GUN); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(data.GUN)
	}

	return r0
}

// GetTargetByName provides a mock function with given fields: name, roles
func (_m *NotaryRepoClient) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	_va := make([]interface{}, len(roles))
	for _i := range roles {
		_va[_i] = roles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *client.TargetWithRole
	if rf, ok := ret.Get(0).(func(string, ...data.RoleName) *client.TargetWithRole); ok {
		r0 = rf(name, roles...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*client.TargetWithRole)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, ...data.RoleName) error); ok {
		r1 = rf(name, roles...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Initialize provides a mock function with given fields: rootKeyIDs, serverManagedRoles
func (_m *NotaryRepoClient) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	_va := make([]interface{}, len(serverManagedRoles))
	for _i := range serverManagedRoles {
		_va[_i] = serverManagedRoles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, rootKeyIDs)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string, ...data.RoleName) error); ok {
		r0 = rf(rootKeyIDs, serverManagedRoles...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InitializeWithCertificate provides a mock function with given fields: rootKeyIDs, rootCerts, serverManagedRoles
func (_m *NotaryRepoClient) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	_va := make([]interface{}, len(serverManagedRoles))
	for _i := range serverManagedRoles {
		_va[_i] = serverManagedRoles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, rootKeyIDs, rootCerts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string, []data.PublicKey, ...data.RoleName) error); ok {
		r0 = rf(rootKeyIDs, rootCerts, serverManagedRoles...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ListRoles provides a mock function with given fields:
func (_m *NotaryRepoClient) ListRoles() ([]client.RoleWithSignatures, error) {
	ret := _m.Called()

	var r0 []client.RoleWithSignatures
	if rf, ok := ret.Get(0).(func() []client.RoleWithSignatures); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]client.RoleWithSignatures)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListTargets provides a mock function with given fields: roles
func (_m *NotaryRepoClient) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	_va := make([]interface{}, len(roles))
	for _i := range roles {
		_va[_i] = roles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []*client.TargetWithRole
	if rf, ok := ret.Get(0).(func(...data.RoleName) []*client.TargetWithRole); ok {
		r0 = rf(roles...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*client.TargetWithRole)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(...data.RoleName) error); ok {
		r1 = rf(roles...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Publish provides a mock function with given fields:
func (_m *NotaryRepoClient) Publish() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveDelegationKeys provides a mock function with given fields: name, keyIDs
func (_m *NotaryRepoClient) RemoveDelegationKeys(name data.RoleName, keyIDs []string) error {
	ret := _m.Called(name, keyIDs)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, []string) error); ok {
		r0 = rf(name, keyIDs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveDelegationKeysAndPaths provides a mock function with given fields: name, keyIDs, paths
func (_m *NotaryRepoClient) RemoveDelegationKeysAndPaths(name data.RoleName, keyIDs []string, paths []string) error {
	ret := _m.Called(name, keyIDs, paths)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, []string, []string) error); ok {
		r0 = rf(name, keyIDs, paths)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveDelegationPaths provides a mock function with given fields: name, paths
func (_m *NotaryRepoClient) RemoveDelegationPaths(name data.RoleName, paths []string) error {
	ret := _m.Called(name, paths)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, []string) error); ok {
		r0 = rf(name, paths)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveDelegationRole provides a mock function with given fields: name
func (_m *NotaryRepoClient) RemoveDelegationRole(name data.RoleName) error {
	ret := _m.Called(name)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName) error); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RemoveTarget provides a mock function with given fields: targetName, roles
func (_m *NotaryRepoClient) RemoveTarget(targetName string, roles ...data.RoleName) error {
	_va := make([]interface{}, len(roles))
	for _i := range roles {
		_va[_i] = roles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, targetName)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, ...data.RoleName) error); ok {
		r0 = rf(targetName, roles...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RotateKey provides a mock function with given fields: role, serverManagesKey, keyList
func (_m *NotaryRepoClient) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	ret := _m.Called(role, serverManagesKey, keyList)

	var r0 error
	if rf, ok := ret.Get(0).(func(data.RoleName, bool, []string) error); ok {
		r0 = rf(role, serverManagesKey, keyList)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetLegacyVersions provides a mock function with given fields: _a0
func (_m *NotaryRepoClient) SetLegacyVersions(_a0 int) {
	_m.Called(_a0)
}

// Witness provides a mock function with given fields: roles
func (_m *NotaryRepoClient) Witness(roles ...data.RoleName) ([]data.RoleName, error) {
	_va := make([]interface{}, len(roles))
	for _i := range roles {
		_va[_i] = roles[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []data.RoleName
	if rf, ok := ret.Get(0).(func(...data.RoleName) []data.RoleName); ok {
		r0 = rf(roles...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]data.RoleName)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(...data.RoleName) error); ok {
		r1 = rf(roles...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewNotaryRepoClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewNotaryRepoClient creates a new instance of NotaryRepoClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNotaryRepoClient(t mockConstructorTestingTNewNotaryRepoClient) *NotaryRepoClient {
	mock := &NotaryRepoClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

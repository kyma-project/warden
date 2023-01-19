package validate

import (
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/client/changelist"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/signed"
	"net"
)

// MOCK NOTARY CLIENT REPOSITORY

type MockNotaryClientRepository struct {
	GetTargetByNameFunc func(name string, roles ...data.RoleName) (*client.TargetWithRole, error)
}

func (m MockNotaryClientRepository) ListTargets(roles ...data.RoleName) ([]*client.TargetWithRole, error) {
	panic("implement me")
}

func (m MockNotaryClientRepository) GetTargetByName(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	return m.GetTargetByNameFunc(name, roles...)
}

func (m MockNotaryClientRepository) GetAllTargetMetadataByName(name string) ([]client.TargetSignedStruct, error) {
	panic("implement me")
}

func (m MockNotaryClientRepository) ListRoles() ([]client.RoleWithSignatures, error) {
	panic("implement me")
}

func (m MockNotaryClientRepository) GetDelegationRoles() ([]data.Role, error) {
	panic("implement me")
}

func (m MockNotaryClientRepository) GetGUN() data.GUN {
	panic("implement me")
}

func (m MockNotaryClientRepository) SetLegacyVersions(i int) {
	panic("implement me")
}

func (m MockNotaryClientRepository) Initialize(rootKeyIDs []string, serverManagedRoles ...data.RoleName) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) InitializeWithCertificate(rootKeyIDs []string, rootCerts []data.PublicKey, serverManagedRoles ...data.RoleName) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) Publish() error {
	panic("implement me")
}

func (m MockNotaryClientRepository) AddTarget(target *client.Target, roles ...data.RoleName) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) RemoveTarget(targetName string, roles ...data.RoleName) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) GetChangelist() (changelist.Changelist, error) {
	panic("implement me")
}

func (m MockNotaryClientRepository) AddDelegation(name data.RoleName, delegationKeys []data.PublicKey, paths []string) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) AddDelegationRoleAndKeys(name data.RoleName, delegationKeys []data.PublicKey) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) AddDelegationPaths(name data.RoleName, paths []string) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) RemoveDelegationKeysAndPaths(name data.RoleName, keyIDs, paths []string) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) RemoveDelegationRole(name data.RoleName) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) RemoveDelegationPaths(name data.RoleName, paths []string) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) RemoveDelegationKeys(name data.RoleName, keyIDs []string) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) ClearDelegationPaths(name data.RoleName) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) Witness(roles ...data.RoleName) ([]data.RoleName, error) {
	panic("implement me")
}

func (m MockNotaryClientRepository) RotateKey(role data.RoleName, serverManagesKey bool, keyList []string) error {
	panic("implement me")
}

func (m MockNotaryClientRepository) GetCryptoService() signed.CryptoService {
	panic("implement me")
}

// MOCK NOTARY REPO FACTORY

type MockNotaryRepoFactory struct {
	GetTargetByNameFunc *func(name string, roles ...data.RoleName) (*client.TargetWithRole, error)
}

func (f MockNotaryRepoFactory) NewRepoClient(img string, c NotaryConfig) (client.Repository, error) {
	r := MockNotaryClientRepository{}
	r.GetTargetByNameFunc = *f.GetTargetByNameFunc
	return r, nil
}

// MOCK NOTARY REPO FACTORY - NO SUCH HOST

type MockNotaryRepoFactoryNoSuchHost struct {
	GetTargetByNameFunc *func(name string, roles ...data.RoleName) (*client.TargetWithRole, error)
}

func (f MockNotaryRepoFactoryNoSuchHost) NewRepoClient(img string, c NotaryConfig) (client.Repository, error) {
	return nil, &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &net.DNSError{
			Err:        "no such host",
			Name:       c.Url,
			IsNotFound: true,
		},
	}
}

// MOCK NOTARY SERVICE BUILDER

type MockNotaryServiceBuilder struct {
	NotaryService notaryService
}

func NewDefaultMockNotaryService() *MockNotaryServiceBuilder {
	f := NewDefaultMockNotaryFunction().Build()
	s := notaryService{
		ServiceConfig: ServiceConfig{
			NotaryConfig: NotaryConfig{},
		},
		RepoFactory: MockNotaryRepoFactory{
			GetTargetByNameFunc: &f,
		},
	}
	return &MockNotaryServiceBuilder{
		NotaryService: s,
	}
}

func (b *MockNotaryServiceBuilder) WithConfig(c NotaryConfig) *MockNotaryServiceBuilder {
	b.NotaryService.NotaryConfig = c
	return b
}

func (b *MockNotaryServiceBuilder) WithRepoFactory(f RepoFactory) *MockNotaryServiceBuilder {
	b.NotaryService.RepoFactory = f
	return b
}

func (b *MockNotaryServiceBuilder) WithFunc(f func(name string, roles ...data.RoleName) (*client.TargetWithRole, error)) *MockNotaryServiceBuilder {
	b.NotaryService.RepoFactory = MockNotaryRepoFactory{
		GetTargetByNameFunc: &f,
	}
	return b
}

func (b *MockNotaryServiceBuilder) WithHash(h []byte) *MockNotaryServiceBuilder {
	f := NewDefaultMockNotaryFunction().WithHash(h).Build()
	b.NotaryService.RepoFactory = MockNotaryRepoFactory{
		GetTargetByNameFunc: &f,
	}
	return b
}

func (b *MockNotaryServiceBuilder) Build() notaryService {
	return b.NotaryService
}

// MOCK NOTARY FUNCTION BUILDER

type MockNotaryFunctionBuilder struct {
	Hash []byte
}

func NewDefaultMockNotaryFunction() *MockNotaryFunctionBuilder {
	return &MockNotaryFunctionBuilder{
		Hash: []byte{1, 2, 3, 4},
	}
}

func (b *MockNotaryFunctionBuilder) WithHash(h []byte) *MockNotaryFunctionBuilder {
	b.Hash = h
	return b
}

func (b *MockNotaryFunctionBuilder) Build() func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
	f := func(name string, roles ...data.RoleName) (*client.TargetWithRole, error) {
		return &client.TargetWithRole{
			Target: client.Target{
				Name:   "ignored",
				Hashes: map[string][]byte{"ignored": b.Hash},
				Length: 1,
			},
		}, nil
	}
	return f
}

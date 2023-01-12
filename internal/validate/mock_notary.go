package validate

import (
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/client/changelist"
	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/tuf/signed"
	"net"
)

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

type MockNotaryRepoFactory struct {
	GetTargetByNameFunc *func(name string, roles ...data.RoleName) (*client.TargetWithRole, error)
}

func (f MockNotaryRepoFactory) NewRepo(img string, c NotaryConfig) (client.Repository, error) {
	r := MockNotaryClientRepository{}
	r.GetTargetByNameFunc = *f.GetTargetByNameFunc
	return r, nil
}

type MockNotaryRepoFactoryNoSuchHost struct {
	GetTargetByNameFunc *func(name string, roles ...data.RoleName) (*client.TargetWithRole, error)
}

func (f MockNotaryRepoFactoryNoSuchHost) NewRepo(img string, c NotaryConfig) (client.Repository, error) {
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

// backend/internal/platform/di/mall/secret_provider_sm.go
package mall

import (
	"context"
	"errors"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var (
	errSecretProviderNotConfigured = errors.New("di.mall: walletSecretProviderSM not configured")
)

// walletSecretProviderSM is used by wiring_policy.go (buildWalletSecretProvider).
// It supports BOTH:
// - usecase.WalletSecretProvider  (brand signer)
// - usecase.AvatarSecretProvider  (avatar signer)
type walletSecretProviderSM struct {
	sm                 *secretmanager.Client
	projectID          string
	brandSecretPrefix  string
	avatarSecretPrefix string
	version            string
}

// Compile-time interface checks.
var _ interface {
	GetBrandSigner(context.Context, string) (any, error)
} = (*walletSecretProviderSM)(nil)
var _ interface {
	GetAvatarSigner(context.Context, string) (any, error)
} = (*walletSecretProviderSM)(nil)

func (p *walletSecretProviderSM) GetBrandSigner(ctx context.Context, brandID string) (any, error) {
	if p == nil || p.sm == nil {
		return nil, errSecretProviderNotConfigured
	}
	bid := brandID
	if bid == "" {
		return nil, errors.New("walletSecretProviderSM: brandID is empty")
	}
	prj := p.projectID
	if prj == "" {
		return nil, errors.New("walletSecretProviderSM: projectID is empty")
	}

	prefix := p.brandSecretPrefix
	if prefix == "" {
		return nil, errors.New("walletSecretProviderSM: brandSecretPrefix is empty")
	}
	ver := p.version
	if ver == "" {
		ver = "latest"
	}

	secretID := prefix + bid
	name := "projects/" + prj + "/secrets/" + secretID + "/versions/" + ver
	resp, err := p.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, errors.New("walletSecretProviderSM: AccessSecretVersion failed (" + name + "): " + err.Error())
	}
	if resp == nil || resp.Payload == nil {
		return nil, errors.New("walletSecretProviderSM: empty payload (" + name + ")")
	}

	return string(resp.Payload.Data), nil
}

func (p *walletSecretProviderSM) GetAvatarSigner(ctx context.Context, avatarID string) (any, error) {
	if p == nil || p.sm == nil {
		return nil, errSecretProviderNotConfigured
	}
	aid := avatarID
	if aid == "" {
		return nil, errors.New("walletSecretProviderSM: avatarID is empty")
	}
	prj := p.projectID
	if prj == "" {
		return nil, errors.New("walletSecretProviderSM: projectID is empty")
	}

	prefix := p.avatarSecretPrefix
	if prefix == "" {
		return nil, errors.New("walletSecretProviderSM: avatarSecretPrefix is empty")
	}
	ver := p.version
	if ver == "" {
		ver = "latest"
	}

	secretID := prefix + aid
	name := "projects/" + prj + "/secrets/" + secretID + "/versions/" + ver
	resp, err := p.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, errors.New("walletSecretProviderSM: AccessSecretVersion failed (" + name + "): " + err.Error())
	}
	if resp == nil || resp.Payload == nil {
		return nil, errors.New("walletSecretProviderSM: empty payload (" + name + ")")
	}

	return string(resp.Payload.Data), nil
}

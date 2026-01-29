// backend/internal/platform/di/mall/secret_provider_sm.go
package mall

import (
	"context"
	"errors"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

var (
	errSecretProviderNotConfigured = errors.New("di.mall: brandWalletSecretProviderSM not configured")
)

// brandWalletSecretProviderSM is used by wiring_policy.go (buildWalletSecretProvider).
// It must remain in package mall.
type brandWalletSecretProviderSM struct {
	sm           *secretmanager.Client
	projectID    string
	secretPrefix string
	version      string
}

func (p *brandWalletSecretProviderSM) GetBrandSigner(ctx context.Context, brandID string) (any, error) {
	if p == nil || p.sm == nil {
		return nil, errSecretProviderNotConfigured
	}
	bid := strings.TrimSpace(brandID)
	if bid == "" {
		return nil, errors.New("brandWalletSecretProviderSM: brandID is empty")
	}
	prj := strings.TrimSpace(p.projectID)
	if prj == "" {
		return nil, errors.New("brandWalletSecretProviderSM: projectID is empty")
	}

	prefix := strings.TrimSpace(p.secretPrefix)
	if prefix == "" {
		return nil, errors.New("brandWalletSecretProviderSM: secretPrefix is empty")
	}
	ver := strings.TrimSpace(p.version)
	if ver == "" {
		ver = "latest"
	}

	secretID := prefix + bid
	name := "projects/" + prj + "/secrets/" + secretID + "/versions/" + ver
	resp, err := p.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, errors.New("brandWalletSecretProviderSM: AccessSecretVersion failed (" + name + "): " + err.Error())
	}
	if resp == nil || resp.Payload == nil {
		return nil, errors.New("brandWalletSecretProviderSM: empty payload (" + name + ")")
	}

	return strings.TrimSpace(string(resp.Payload.Data)), nil
}

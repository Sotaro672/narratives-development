package mallHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "cloud.google.com/go/iam/credentials/apiv1/credentialspb"
)

type serviceAccountKey struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

func loadServiceAccountKey(filepath string) (email string, privateKey []byte, err error) {
	bs, err := os.ReadFile(filepath)
	if err != nil {
		return "", nil, err
	}
	var k serviceAccountKey
	if err := json.Unmarshal(bs, &k); err != nil {
		return "", nil, err
	}
	e := strings.TrimSpace(k.ClientEmail)
	pk := strings.TrimSpace(k.PrivateKey)
	if e == "" || pk == "" {
		return "", nil, fmt.Errorf("missing client_email/private_key in credentials")
	}
	return e, []byte(pk), nil
}

// signBytesWithIAM signs bytes via IAM Credentials API SignBlob.
func signBytesWithIAM(ctx context.Context, signerEmail string, payload []byte) ([]byte, error) {
	c, err := credentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	name := fmt.Sprintf("projects/-/serviceAccounts/%s", signerEmail)
	resp, err := c.SignBlob(ctx, &credentialspb.SignBlobRequest{
		Name:    name,
		Payload: payload,
	})
	if err != nil {
		return nil, err
	}
	return resp.SignedBlob, nil
}

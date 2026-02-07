// backend/internal/adapters/in/http/mall/handler/avatar/helpers_iam_signer.go
package avatarHandler

import (
	"context"
	"fmt"

	credentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "cloud.google.com/go/iam/credentials/apiv1/credentialspb"
)

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

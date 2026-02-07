// backend/internal/adapters/in/http/mall/handler/avatar/helpers_gcs_signer.go
package avatarHandler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

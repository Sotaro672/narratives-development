// backend/internal/application/query/sns/errors.go
package sns

import "errors"

// ErrNotFound is a shared sentinel error for "not found" in sns query package.
// - handlers may check with errors.Is(err, sns.ErrNotFound)
var ErrNotFound = errors.New("not_found")

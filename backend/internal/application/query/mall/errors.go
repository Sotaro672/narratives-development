// backend\internal\application\query\mall\errors.go
package mall

import "errors"

// ErrNotFound is a shared sentinel error for "not found" in sns query package.
// - handlers may check with errors.Is(err, sns.ErrNotFound)
var ErrNotFound = errors.New("not_found")

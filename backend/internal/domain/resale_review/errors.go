// backend/internal/domain/resale_review/errors.go
package resale_review

import (
	"errors"
	"fmt"
)

// ErrCommentAlreadyRead indicates that a comment cannot be deleted because
// the resale owner has already read it.
//
// This error wraps ErrConflict so existing HTTP error mapping through
// IsConflict continues to return HTTP 409 Conflict.
var ErrCommentAlreadyRead = fmt.Errorf(
	"%w: comment already read",
	ErrConflict,
)

// IsCommentAlreadyRead reports whether err represents an attempt to delete
// a comment that has already been read by the resale owner.
func IsCommentAlreadyRead(err error) bool {
	return errors.Is(err, ErrCommentAlreadyRead)
}

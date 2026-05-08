// backend/internal/domain/tokenBlueprint_review/entity.go
package tokenBlueprint_review

import (
	"errors"
	"fmt"
	"time"
)

// ============================================================
// Concept (Firestore mapping idea; repository / application.usecase responsibility)
//
// tokenBlueprintReviews/{tokenBlueprintId}
//   - likeCount
//   - dislikeCount
//   - topLevelCommentCount
//   - totalCommentCount
//   - pinnedCommentId
//   - createdAt
//   - updatedAt
//
//   - reactions/{actorType_actorId} -> TokenBlueprintReaction
//
//   - comments/{commentId} -> Comment
//       - commentId
//       - tokenBlueprintId
//       - parentCommentId   // "" if top-level comment
//       - rootCommentId     // top-level ancestor comment id
//       - depth             // 0 = top-level, 1 = reply, 2 = reply to reply...
//       - authorId
//       - authorType
//       - isOwnerComment    // true when author brand == tokenBlueprint owner brand
//       - body
//       - likeCount
//       - dislikeCount
//       - childCount        // direct children count
//       - deleted
//       - createdAt
//       - updatedAt
//
//       - reactions/{actorType_actorId} -> CommentReaction
//
// NOTE:
// - This file is domain-only. No Firestore / authentication / authorization code here.
// - Whether mall can post only as avatar, or console can post only as brand,
//   must be controlled by application.usecase.
// - Whether a brand is the tokenBlueprint owner and therefore allowed to create
//   owner comments / replies must also be determined by application.usecase.
// ============================================================

// ---------------------------
// Errors
// ---------------------------

var (
	ErrInvalidID           = errors.New("invalid id")
	ErrInvalidReactionType = errors.New("invalid reaction type")
	ErrInvalidAuthorType   = errors.New("invalid author type")
	ErrInvalidActorType    = errors.New("invalid actor type")
	ErrNegativeCounter     = errors.New("counter would become negative")
	ErrEmptyBody           = errors.New("body must not be empty")
	ErrInvalidParent       = errors.New("invalid parent relation")
	ErrDeletedComment      = errors.New("comment is deleted")
	ErrInvalidDepth        = errors.New("invalid depth")
)

// ---------------------------
// ReactionType (Comment/Like/Dislike)
// ---------------------------

type ReactionType string

const (
	ReactionComment ReactionType = "comment"
	ReactionLike    ReactionType = "like"
	ReactionDislike ReactionType = "dislike"
)

func (t ReactionType) Validate() error {
	switch t {
	case ReactionComment, ReactionLike, ReactionDislike:
		return nil
	default:
		return ErrInvalidReactionType
	}
}

// ---------------------------
// AuthorType (Avatar/Brand)
// ---------------------------

type AuthorType string

const (
	AuthorTypeAvatar AuthorType = "avatar"
	AuthorTypeBrand  AuthorType = "brand"
)

func (t AuthorType) Validate() error {
	switch t {
	case AuthorTypeAvatar, AuthorTypeBrand:
		return nil
	default:
		return ErrInvalidAuthorType
	}
}

// ---------------------------
// ActorType (Reaction actor: Avatar/Brand)
// ---------------------------

type ActorType string

const (
	ActorTypeAvatar ActorType = "avatar"
	ActorTypeBrand  ActorType = "brand"
)

func (t ActorType) Validate() error {
	switch t {
	case ActorTypeAvatar, ActorTypeBrand:
		return nil
	default:
		return ErrInvalidActorType
	}
}

// NextReactionType implements YouTube-like toggle behavior BUT with 3-state domain:
// comment / like / dislike.
//
// Interpretation (since "none" is removed):
// - ReactionComment is treated as the neutral/default state (i.e., "no like/dislike").
//
// Rules:
// - current=like    + pressed=like    => comment
// - current=like    + pressed=dislike => dislike
// - current=dislike + pressed=dislike => comment
// - current=dislike + pressed=like    => like
// - current=comment + pressed=like    => like
// - current=comment + pressed=dislike => dislike
// - pressed=comment => comment (explicit clear / reset)
func NextReactionType(current, pressed ReactionType) (ReactionType, error) {
	if err := current.Validate(); err != nil {
		return "", err
	}
	if err := pressed.Validate(); err != nil {
		return "", err
	}

	if pressed == ReactionComment {
		return ReactionComment, nil
	}
	if current == pressed {
		return ReactionComment, nil
	}
	return pressed, nil
}

// reactionDelta returns like/dislike counter delta between old -> new.
// Only like/dislike affect counters.
func reactionDelta(oldT, newT ReactionType) (likeDelta int64, dislikeDelta int64) {
	if oldT == newT {
		return 0, 0
	}

	switch oldT {
	case ReactionLike:
		likeDelta -= 1
	case ReactionDislike:
		dislikeDelta -= 1
	}

	switch newT {
	case ReactionLike:
		likeDelta += 1
	case ReactionDislike:
		dislikeDelta += 1
	}

	return likeDelta, dislikeDelta
}

// ---------------------------
// TokenBlueprint aggregate (top-level)
// ---------------------------

// TokenBlueprintReviewAggregate represents the parent document:
// tokenBlueprintReviews/{tokenBlueprintId}
type TokenBlueprintReviewAggregate struct {
	TokenBlueprintID     string
	LikeCount            int64
	DislikeCount         int64
	TopLevelCommentCount int64
	TotalCommentCount    int64
	PinnedCommentID      string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewTokenBlueprintReviewAggregate(tokenBlueprintID string, now time.Time) (*TokenBlueprintReviewAggregate, error) {
	if tokenBlueprintID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	return &TokenBlueprintReviewAggregate{
		TokenBlueprintID:     tokenBlueprintID,
		LikeCount:            0,
		DislikeCount:         0,
		TopLevelCommentCount: 0,
		TotalCommentCount:    0,
		PinnedCommentID:      "",
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

// ApplyReaction changes counters based on actor's reaction change (old -> new).
// The per-actor reaction document itself should be stored in subcollection by repository layer.
func (a *TokenBlueprintReviewAggregate) ApplyReaction(oldType, newType ReactionType, now time.Time) error {
	if err := oldType.Validate(); err != nil {
		return err
	}
	if err := newType.Validate(); err != nil {
		return err
	}

	ld, dd := reactionDelta(oldType, newType)

	if a.LikeCount+ld < 0 || a.DislikeCount+dd < 0 {
		return ErrNegativeCounter
	}

	a.LikeCount += ld
	a.DislikeCount += dd
	a.UpdatedAt = now
	return nil
}

func (a *TokenBlueprintReviewAggregate) IncrementTopLevelCommentCount(now time.Time) {
	a.TopLevelCommentCount += 1
	a.UpdatedAt = now
}

func (a *TokenBlueprintReviewAggregate) DecrementTopLevelCommentCount(now time.Time) error {
	if a.TopLevelCommentCount-1 < 0 {
		return ErrNegativeCounter
	}
	a.TopLevelCommentCount -= 1
	a.UpdatedAt = now
	return nil
}

func (a *TokenBlueprintReviewAggregate) IncrementTotalCommentCount(now time.Time) {
	a.TotalCommentCount += 1
	a.UpdatedAt = now
}

func (a *TokenBlueprintReviewAggregate) DecrementTotalCommentCount(now time.Time) error {
	if a.TotalCommentCount-1 < 0 {
		return ErrNegativeCounter
	}
	a.TotalCommentCount -= 1
	a.UpdatedAt = now
	return nil
}

// PinComment / UnpinComment are domain-safe setters.
// Whether the caller is allowed to pin is application.usecase responsibility.
func (a *TokenBlueprintReviewAggregate) PinComment(commentID string, now time.Time) error {
	if commentID == "" {
		return fmt.Errorf("%w: commentID", ErrInvalidID)
	}
	a.PinnedCommentID = commentID
	a.UpdatedAt = now
	return nil
}

func (a *TokenBlueprintReviewAggregate) UnpinComment(now time.Time) {
	a.PinnedCommentID = ""
	a.UpdatedAt = now
}

// ---------------------------
// Reaction docs (subcollection items)
// ---------------------------

// TokenBlueprintReaction represents:
// tokenBlueprintReviews/{tokenBlueprintId}/reactions/{actorType_actorId}
type TokenBlueprintReaction struct {
	TokenBlueprintID string
	ActorID          string
	ActorType        ActorType
	Type             ReactionType
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewTokenBlueprintReaction(tokenBlueprintID, actorID string, actorType ActorType, t ReactionType, now time.Time) (*TokenBlueprintReaction, error) {
	if tokenBlueprintID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if actorID == "" {
		return nil, fmt.Errorf("%w: actorID", ErrInvalidID)
	}
	if err := actorType.Validate(); err != nil {
		return nil, err
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &TokenBlueprintReaction{
		TokenBlueprintID: tokenBlueprintID,
		ActorID:          actorID,
		ActorType:        actorType,
		Type:             t,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (r *TokenBlueprintReaction) ChangeType(newT ReactionType, now time.Time) error {
	if err := newT.Validate(); err != nil {
		return err
	}
	r.Type = newT
	r.UpdatedAt = now
	return nil
}

// ReactionDocumentID returns a stable subcollection document id.
// Repository can use this helper, but persistence itself is repository responsibility.
func (r *TokenBlueprintReaction) ReactionDocumentID() (string, error) {
	if r.ActorID == "" {
		return "", fmt.Errorf("%w: actorID", ErrInvalidID)
	}
	if err := r.ActorType.Validate(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", r.ActorType, r.ActorID), nil
}

// ---------------------------
// Comment (top-level comment / reply / reply-to-reply / ...)
// ---------------------------

// Comment represents:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}
type Comment struct {
	CommentID        string
	TokenBlueprintID string
	ParentCommentID  string
	RootCommentID    string
	Depth            int

	AuthorID       string
	AuthorType     AuthorType
	IsOwnerComment bool
	Body           string

	LikeCount    int64
	DislikeCount int64
	ChildCount   int64

	Deleted bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTopLevelComment creates a top-level comment.
// Rules:
// - ParentCommentID = ""
// - RootCommentID   = CommentID
// - Depth           = 0
//
// isOwnerComment should be decided by application.usecase.
// Example:
// - avatar comment from mall     => false
// - brand comment from console   => true only when brand is tokenBlueprint owner
func NewTopLevelComment(
	commentID,
	tokenBlueprintID,
	authorID string,
	authorType AuthorType,
	isOwnerComment bool,
	body string,
	now time.Time,
) (*Comment, error) {
	if commentID == "" {
		return nil, fmt.Errorf("%w: commentID", ErrInvalidID)
	}
	if tokenBlueprintID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if authorID == "" {
		return nil, fmt.Errorf("%w: authorID", ErrInvalidID)
	}
	if err := authorType.Validate(); err != nil {
		return nil, err
	}
	if body == "" {
		return nil, ErrEmptyBody
	}

	return &Comment{
		CommentID:        commentID,
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  "",
		RootCommentID:    commentID,
		Depth:            0,
		AuthorID:         authorID,
		AuthorType:       authorType,
		IsOwnerComment:   isOwnerComment,
		Body:             body,
		LikeCount:        0,
		DislikeCount:     0,
		ChildCount:       0,
		Deleted:          false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// NewReplyComment creates a reply to any existing comment.
// This supports infinite nesting.
//
// Rules:
// - ParentCommentID = parent.CommentID
// - RootCommentID   = parent.RootCommentID
// - Depth           = parent.Depth + 1
//
// isOwnerComment should be decided by application.usecase.
func NewReplyComment(
	commentID,
	tokenBlueprintID string,
	parent *Comment,
	authorID string,
	authorType AuthorType,
	isOwnerComment bool,
	body string,
	now time.Time,
) (*Comment, error) {
	if commentID == "" {
		return nil, fmt.Errorf("%w: commentID", ErrInvalidID)
	}
	if tokenBlueprintID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if parent == nil {
		return nil, ErrInvalidParent
	}
	if parent.TokenBlueprintID != tokenBlueprintID {
		return nil, ErrInvalidParent
	}
	if parent.Deleted {
		return nil, ErrDeletedComment
	}
	if authorID == "" {
		return nil, fmt.Errorf("%w: authorID", ErrInvalidID)
	}
	if err := authorType.Validate(); err != nil {
		return nil, err
	}
	if body == "" {
		return nil, ErrEmptyBody
	}
	if parent.Depth < 0 {
		return nil, ErrInvalidDepth
	}
	if parent.RootCommentID == "" {
		return nil, fmt.Errorf("%w: parent.RootCommentID", ErrInvalidID)
	}

	return &Comment{
		CommentID:        commentID,
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  parent.CommentID,
		RootCommentID:    parent.RootCommentID,
		Depth:            parent.Depth + 1,
		AuthorID:         authorID,
		AuthorType:       authorType,
		IsOwnerComment:   isOwnerComment,
		Body:             body,
		LikeCount:        0,
		DislikeCount:     0,
		ChildCount:       0,
		Deleted:          false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (c *Comment) IsTopLevel() bool {
	return c.ParentCommentID == "" && c.Depth == 0 && c.RootCommentID == c.CommentID
}

func (c *Comment) UpdateBody(body string, now time.Time) error {
	if body == "" {
		return ErrEmptyBody
	}
	if c.Deleted {
		return ErrDeletedComment
	}
	c.Body = body
	c.UpdatedAt = now
	return nil
}

// MarkDeleted keeps the comment node but clears visible content.
// Existing descendants can remain attached to this node.
func (c *Comment) MarkDeleted(now time.Time) {
	c.Deleted = true
	c.Body = ""
	c.UpdatedAt = now
}

// ApplyReaction changes counters based on actor's reaction change (old -> new).
// The per-actor reaction document is stored under:
// comments/{commentId}/reactions/{actorType_actorId}
func (c *Comment) ApplyReaction(oldType, newType ReactionType, now time.Time) error {
	if err := oldType.Validate(); err != nil {
		return err
	}
	if err := newType.Validate(); err != nil {
		return err
	}

	ld, dd := reactionDelta(oldType, newType)
	if c.LikeCount+ld < 0 || c.DislikeCount+dd < 0 {
		return ErrNegativeCounter
	}

	c.LikeCount += ld
	c.DislikeCount += dd
	c.UpdatedAt = now
	return nil
}

func (c *Comment) IncrementChildCount(now time.Time) {
	c.ChildCount += 1
	c.UpdatedAt = now
}

func (c *Comment) DecrementChildCount(now time.Time) error {
	if c.ChildCount-1 < 0 {
		return ErrNegativeCounter
	}
	c.ChildCount -= 1
	c.UpdatedAt = now
	return nil
}

// ---------------------------
// CommentReaction
// ---------------------------

// CommentReaction represents:
// tokenBlueprintReviews/{tokenBlueprintId}/comments/{commentId}/reactions/{actorType_actorId}
type CommentReaction struct {
	TokenBlueprintID string
	CommentID        string
	ActorID          string
	ActorType        ActorType
	Type             ReactionType
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewCommentReaction(tokenBlueprintID, commentID, actorID string, actorType ActorType, t ReactionType, now time.Time) (*CommentReaction, error) {
	if tokenBlueprintID == "" {
		return nil, fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if commentID == "" {
		return nil, fmt.Errorf("%w: commentID", ErrInvalidID)
	}
	if actorID == "" {
		return nil, fmt.Errorf("%w: actorID", ErrInvalidID)
	}
	if err := actorType.Validate(); err != nil {
		return nil, err
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}

	return &CommentReaction{
		TokenBlueprintID: tokenBlueprintID,
		CommentID:        commentID,
		ActorID:          actorID,
		ActorType:        actorType,
		Type:             t,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (r *CommentReaction) ChangeType(newT ReactionType, now time.Time) error {
	if err := newT.Validate(); err != nil {
		return err
	}
	r.Type = newT
	r.UpdatedAt = now
	return nil
}

// ReactionDocumentID returns a stable subcollection document id.
// Repository can use this helper, but persistence itself is repository responsibility.
func (r *CommentReaction) ReactionDocumentID() (string, error) {
	if r.ActorID == "" {
		return "", fmt.Errorf("%w: actorID", ErrInvalidID)
	}
	if err := r.ActorType.Validate(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", r.ActorType, r.ActorID), nil
}

// ---------------------------
// Optional: sanity helpers for repository / application.usecase layer
// ---------------------------

// ValidateComment ensures structural invariants for a single comment.
func ValidateComment(c *Comment) error {
	if c == nil {
		return ErrInvalidParent
	}
	if c.CommentID == "" {
		return fmt.Errorf("%w: commentID", ErrInvalidID)
	}
	if c.TokenBlueprintID == "" {
		return fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if c.AuthorID == "" {
		return fmt.Errorf("%w: authorID", ErrInvalidID)
	}
	if err := c.AuthorType.Validate(); err != nil {
		return err
	}
	if c.Depth < 0 {
		return ErrInvalidDepth
	}
	if c.RootCommentID == "" {
		return fmt.Errorf("%w: rootCommentID", ErrInvalidID)
	}

	if c.IsTopLevel() {
		return nil
	}

	if c.ParentCommentID == "" {
		return ErrInvalidParent
	}
	if c.Depth == 0 {
		return ErrInvalidDepth
	}
	if c.RootCommentID == c.CommentID {
		return ErrInvalidParent
	}

	return nil
}

// ValidateCommentParentRelation ensures child belongs to the given parent and tokenBlueprint.
func ValidateCommentParentRelation(tokenBlueprintID string, parent, child *Comment) error {
	if parent == nil || child == nil {
		return ErrInvalidParent
	}
	if err := ValidateComment(parent); err != nil {
		return err
	}
	if err := ValidateComment(child); err != nil {
		return err
	}
	if parent.TokenBlueprintID != tokenBlueprintID {
		return ErrInvalidParent
	}
	if child.TokenBlueprintID != tokenBlueprintID {
		return ErrInvalidParent
	}
	if child.ParentCommentID != parent.CommentID {
		return ErrInvalidParent
	}
	if child.Depth != parent.Depth+1 {
		return ErrInvalidParent
	}

	expectedRootCommentID := parent.RootCommentID
	if parent.IsTopLevel() {
		expectedRootCommentID = parent.CommentID
	}
	if child.RootCommentID != expectedRootCommentID {
		return ErrInvalidParent
	}

	return nil
}

func ValidateTokenBlueprintReaction(r *TokenBlueprintReaction) error {
	if r == nil {
		return ErrInvalidParent
	}
	if r.TokenBlueprintID == "" {
		return fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if r.ActorID == "" {
		return fmt.Errorf("%w: actorID", ErrInvalidID)
	}
	if err := r.ActorType.Validate(); err != nil {
		return err
	}
	if err := r.Type.Validate(); err != nil {
		return err
	}
	return nil
}

func ValidateCommentReaction(r *CommentReaction) error {
	if r == nil {
		return ErrInvalidParent
	}
	if r.TokenBlueprintID == "" {
		return fmt.Errorf("%w: tokenBlueprintID", ErrInvalidID)
	}
	if r.CommentID == "" {
		return fmt.Errorf("%w: commentID", ErrInvalidID)
	}
	if r.ActorID == "" {
		return fmt.Errorf("%w: actorID", ErrInvalidID)
	}
	if err := r.ActorType.Validate(); err != nil {
		return err
	}
	if err := r.Type.Validate(); err != nil {
		return err
	}
	return nil
}

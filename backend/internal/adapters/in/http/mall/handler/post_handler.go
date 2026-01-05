// backend/internal/adapters/in/http/sns/handler/post_handler.go
package mallHandler

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	postimagedom "narratives/internal/domain/postImage"
)

// PostHandler serves SNS post-related endpoints (currently: post images on GCS).
//
// Intended mount examples (router side):
// - POST   /sns/posts/prefix                 (ensure GCS prefix placeholder)
// - POST   /sns/posts/images                 (multipart upload -> GCS Put)
// - GET    /sns/posts/images                 (list objects under prefix)
// - DELETE /sns/posts/images                 (delete object or by prefix)
//
// Storage policy (single bucket):
// - bucket: narratives-development-posts
// - objectPath: avatars/{avatarId}/posts/{postId}/{fileName}
//
// NOTE:
//   - This handler does NOT persist metadata into Firestore yet.
//     It only uploads/returns public URL(s) assuming the bucket is public.
type PostHandler struct {
	store postimagedom.ObjectStoragePort
}

func NewPostHandler(store postimagedom.ObjectStoragePort) http.Handler {
	return &PostHandler{store: store}
}

func (h *PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		writeErr(w, http.StatusInternalServerError, "post handler is not configured")
		return
	}

	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	// ====== POST /sns/posts/prefix
	case strings.HasSuffix(path, "/sns/posts/prefix") && r.Method == http.MethodPost:
		h.handleEnsurePrefix(w, r)
		return

	// ====== POST /sns/posts/images  (multipart upload)
	case strings.HasSuffix(path, "/sns/posts/images") && r.Method == http.MethodPost:
		h.handleUploadPostImage(w, r)
		return

	// ====== GET /sns/posts/images
	case strings.HasSuffix(path, "/sns/posts/images") && r.Method == http.MethodGet:
		h.handleListPostImages(w, r)
		return

	// ====== DELETE /sns/posts/images
	case strings.HasSuffix(path, "/sns/posts/images") && r.Method == http.MethodDelete:
		h.handleDeletePostImages(w, r)
		return
	}

	writeErr(w, http.StatusNotFound, "not found")
}

// --------------------------------------------------
// POST /sns/posts/prefix
// --------------------------------------------------

type ensurePrefixReq struct {
	AvatarID string `json:"avatarId"`
	PostID   string `json:"postId,omitempty"`
}

func (h *PostHandler) handleEnsurePrefix(w http.ResponseWriter, r *http.Request) {
	var req ensurePrefixReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return
	}

	aid := strings.TrimSpace(req.AvatarID)
	if aid == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}

	// Ensure "avatars/<avatarId>/posts/" or "avatars/<avatarId>/posts/<postId>/"
	prefix := "avatars/" + aid + "/posts/"
	if pid := strings.TrimSpace(req.PostID); pid != "" {
		prefix = "avatars/" + aid + "/posts/" + pid + "/"
	}

	if err := h.store.EnsurePrefix(r.Context(), postimagedom.DefaultBucket, prefix); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"bucket": postimagedom.DefaultBucket,
		"prefix": prefix,
	})
}

// --------------------------------------------------
// POST /sns/posts/images (multipart upload)
// --------------------------------------------------

const (
	// keep it conservative for web uploads; adjust later if needed
	maxPostImageBytes = 20 << 20 // 20MB
)

func (h *PostHandler) handleUploadPostImage(w http.ResponseWriter, r *http.Request) {
	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxPostImageBytes)

	// Parse multipart
	if err := r.ParseMultipartForm(maxPostImageBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	aid := strings.TrimSpace(r.FormValue("avatarId"))
	pid := strings.TrimSpace(r.FormValue("postId"))
	if aid == "" || pid == "" {
		writeErr(w, http.StatusBadRequest, "avatarId and postId are required")
		return
	}

	f, fh, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "file is required")
		return
	}
	defer f.Close()

	fileName := strings.TrimSpace(fh.Filename)
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		writeErr(w, http.StatusBadRequest, "invalid fileName")
		return
	}

	// Read bytes (bounded by MaxBytesReader)
	data, err := io.ReadAll(f)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "failed to read file")
		return
	}
	if len(data) == 0 {
		writeErr(w, http.StatusBadRequest, "file is empty")
		return
	}

	ct := strings.TrimSpace(fh.Header.Get("Content-Type"))
	if ct == "" {
		ct = strings.TrimSpace(r.FormValue("contentType"))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}

	// Ensure prefix exists for console UX
	_ = h.store.EnsurePrefix(r.Context(), postimagedom.DefaultBucket, "avatars/"+aid+"/posts/"+pid+"/")

	objectPath := postimagedom.BuildObjectPath(aid, pid, fileName)

	if err := h.store.Put(r.Context(), postimagedom.DefaultBucket, objectPath, ct, data); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	now := time.Now().UTC()

	pi, err := postimagedom.NewPublicPostImage(newID(), aid, pid, fileName, now)
	if err != nil {
		// upload succeeded but metadata build failed -> still return URL best-effort
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         true,
			"bucket":     postimagedom.DefaultBucket,
			"objectPath": objectPath,
			"url":        h.store.PublicURL(postimagedom.DefaultBucket, objectPath),
		})
		return
	}

	// attach optional metadata
	fn := fileName
	pi.FileName = &fn
	cts := ct
	pi.ContentType = &cts
	sz := int64(len(data))
	pi.Size = &sz

	writeJSON(w, http.StatusOK, toPostImageResponse(pi, h.store))
}

// --------------------------------------------------
// GET /sns/posts/images
// --------------------------------------------------

func (h *PostHandler) handleListPostImages(w http.ResponseWriter, r *http.Request) {
	aid := strings.TrimSpace(r.URL.Query().Get("avatarId"))
	if aid == "" {
		writeErr(w, http.StatusBadRequest, "avatarId is required")
		return
	}
	pid := strings.TrimSpace(r.URL.Query().Get("postId"))

	prefix := "avatars/" + aid + "/posts/"
	if pid != "" {
		prefix = "avatars/" + aid + "/posts/" + pid + "/"
	}

	paths, err := h.store.ListObjectPaths(r.Context(), postimagedom.DefaultBucket, prefix)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	// stable
	sort.Strings(paths)

	type item struct {
		ObjectPath string `json:"objectPath"`
		URL        string `json:"url"`
	}

	out := make([]item, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, item{
			ObjectPath: p,
			URL:        h.store.PublicURL(postimagedom.DefaultBucket, p),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"bucket": postimagedom.DefaultBucket,
		"prefix": prefix,
		"items":  out,
	})
}

// --------------------------------------------------
// DELETE /sns/posts/images
// --------------------------------------------------

func (h *PostHandler) handleDeletePostImages(w http.ResponseWriter, r *http.Request) {
	// Policy:
	// - if objectPath is given -> delete single object
	// - else if avatarId+postId given -> delete prefix
	objectPath := strings.TrimSpace(r.URL.Query().Get("objectPath"))
	if objectPath != "" {
		ops := []postimagedom.GCSDeleteOp{{
			Bucket:     postimagedom.DefaultBucket,
			ObjectPath: objectPath,
		}}
		if err := h.store.DeleteObjects(r.Context(), ops); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	aid := strings.TrimSpace(r.URL.Query().Get("avatarId"))
	pid := strings.TrimSpace(r.URL.Query().Get("postId"))
	if aid == "" || pid == "" {
		writeErr(w, http.StatusBadRequest, "objectPath OR (avatarId and postId) are required")
		return
	}

	prefix := "avatars/" + aid + "/posts/" + pid + "/"
	if err := h.store.DeleteByPrefix(r.Context(), postimagedom.DefaultBucket, prefix); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --------------------------------------------------
// response DTO
// --------------------------------------------------

type postImageResponse struct {
	ID          string  `json:"id"`
	AvatarID    string  `json:"avatarId"`
	Bucket      string  `json:"bucket"`
	ObjectPath  string  `json:"objectPath"`
	URL         string  `json:"url"`
	FileName    *string `json:"fileName,omitempty"`
	ContentType *string `json:"contentType,omitempty"`
	Size        *int64  `json:"size,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

func toPostImageResponse(p postimagedom.PostImage, store postimagedom.ObjectStoragePort) postImageResponse {
	return postImageResponse{
		ID:          strings.TrimSpace(p.ID),
		AvatarID:    strings.TrimSpace(p.AvatarID),
		Bucket:      strings.TrimSpace(p.Bucket),
		ObjectPath:  strings.TrimSpace(p.ObjectPath),
		URL:         store.PublicURL(p.Bucket, p.ObjectPath),
		FileName:    p.FileName,
		ContentType: p.ContentType,
		Size:        p.Size,
		CreatedAt:   toRFC3339(p.CreatedAt),
	}
}

// --------------------------------------------------
// local helpers
// --------------------------------------------------

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback: timestamp-based
		return "pi_" + time.Now().UTC().Format("20060102T150405.000000000Z")
	}
	return hex.EncodeToString(b[:])
}

// sanitizeFileName removes any path fragments and trims.
// (same behavior as domain helper; duplicated locally to avoid relying on unexported funcs)
func sanitizeFileName(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "\\", "/")
	if i := strings.LastIndex(v, "/"); i >= 0 {
		v = v[i+1:]
	}
	v = strings.TrimSpace(v)
	if v == "" || v == "." || v == ".." {
		return ""
	}
	// forbid query-like tails
	if strings.Contains(v, "?") || strings.Contains(v, "#") {
		return ""
	}
	return v
}

/*
NOTE:
- This file relies on shared helpers already used by other handlers in this package:
  - writeJSON
  - writeErr
  - jsonNewDecoder (or equivalent)
If your handler package does not yet have jsonNewDecoder, replace readJSON with your existing helper.
*/

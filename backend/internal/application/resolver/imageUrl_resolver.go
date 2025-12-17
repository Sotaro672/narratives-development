package resolver

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

// ImageURLResolver resolves/normalizes image URLs for storage and response.
//
// 想定ユースケース:
// - Frontend から "https://storage.googleapis.com/<bucket>/<objectPath>" を受け取る
// - Backend では保存用に "<objectPath>" を保存（例: tokenBlueprint.iconId）
// - レスポンスでは iconUrl が空なら、bucket + objectPath から public URL を生成して返す
//
// 重要:
// - Request Header ではなく Body で URL を送る（日本語ファイル名などで ISO-8859-1 問題を回避）
// - 署名付き URL（クエリ付き）でも objectPath だけを抽出して保存可能
type ImageURLResolver struct {
	// PublicBucketName is the GCS bucket name used to build public URL when missing in response.
	// 例: narratives-development_token_icon
	PublicBucketName string
}

// NewImageURLResolver creates resolver.
// env優先:
// - TOKEN_ICON_PUBLIC_BUCKET
// 引数 bucket が空なら env / fallback を使う。
func NewImageURLResolver(bucket string) *ImageURLResolver {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = strings.TrimSpace(os.Getenv("TOKEN_ICON_PUBLIC_BUCKET"))
	}
	// 最後の fallback は空でもOK（この場合は build public URL しない）
	return &ImageURLResolver{PublicBucketName: b}
}

// ResolveForSave:
// - rawURL: frontend が送ってくる画像 URL（または objectPath っぽい文字列）
// 戻り値:
// - objectPath: 保存に使う正規化パス（例: "i5GmRz2FXOEWL5lTuY5F/icon"）
// - publicURL: 返却に使う正規化 public URL（bucket が取れる/設定されている場合）
// 注意:
// - rawURL が空なら全部空で返す（未設定扱い）
func (r *ImageURLResolver) ResolveForSave(rawURL string) (objectPath string, publicURL string, err error) {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return "", "", nil
	}

	// 1) URL っぽいなら parse して bucket/object を抽出
	if looksLikeURL(raw) {
		bkt, obj, e := parseGCSLikeURL(raw)
		if e != nil {
			return "", "", e
		}
		obj = normalizeObjectPath(obj)
		if obj == "" {
			return "", "", errors.New("objectPath is empty after normalize")
		}

		// 保存は objectPath だけ
		objectPath = obj

		// 返却は bucket が URL に含まれていればそれを優先、無ければ resolver.bucket を使う
		if bkt == "" {
			bkt = strings.TrimSpace(r.PublicBucketName)
		}
		if bkt != "" {
			publicURL = buildGCSPublicURL(bkt, objectPath)
		}
		return objectPath, publicURL, nil
	}

	// 2) URL じゃない場合は「objectPath とみなす」
	obj := normalizeObjectPath(raw)
	if obj == "" {
		return "", "", errors.New("objectPath is empty")
	}
	objectPath = obj

	bkt := strings.TrimSpace(r.PublicBucketName)
	if bkt != "" {
		publicURL = buildGCSPublicURL(bkt, objectPath)
	}

	return objectPath, publicURL, nil
}

// ResolveForResponse:
// - storedObjectPath: DB に保存されている objectPath（iconId 等）
// - storedIconURL: DB に iconUrl を保存している場合の値（通常は空でもOK）
// 戻り値:
// - iconUrl: フロントへ返す URL（iconUrl が空なら objectPath + bucket から生成）
func (r *ImageURLResolver) ResolveForResponse(storedObjectPath string, storedIconURL string) string {
	u := strings.TrimSpace(storedIconURL)
	if u != "" {
		// 既に URL が入ってるならそれを返す（必要ならここで正規化も可）
		return u
	}

	obj := normalizeObjectPath(storedObjectPath)
	if obj == "" {
		return ""
	}

	bkt := strings.TrimSpace(r.PublicBucketName)
	if bkt == "" {
		return ""
	}

	return buildGCSPublicURL(bkt, obj)
}

// ------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------

func looksLikeURL(s string) bool {
	ls := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://") || strings.HasPrefix(ls, "gs://")
}

// parseGCSLikeURL supports:
//
// 1) https://storage.googleapis.com/<bucket>/<objectPath>?X-Goog-...
// 2) https://<bucket>.storage.googleapis.com/<objectPath>?X-Goog-...
// 3) gs://<bucket>/<objectPath>
//
// Returns: bucket, objectPath
func parseGCSLikeURL(raw string) (bucket string, objectPath string, err error) {
	s := strings.TrimSpace(raw)

	// gs://bucket/object
	if strings.HasPrefix(strings.ToLower(s), "gs://") {
		trim := strings.TrimPrefix(s, "gs://")
		trim = strings.TrimPrefix(trim, "GS://")

		parts := strings.SplitN(trim, "/", 2)
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			return "", "", errors.New("invalid gs:// url: bucket is empty")
		}
		bucket = strings.TrimSpace(parts[0])
		if len(parts) == 1 {
			return bucket, "", errors.New("invalid gs:// url: objectPath is empty")
		}
		objectPath = parts[1]
		return bucket, objectPath, nil
	}

	u, e := url.Parse(s)
	if e != nil {
		return "", "", e
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	path := u.Path // クエリは無視（署名付きでもOK）

	// https://storage.googleapis.com/bucket/object
	if host == "storage.googleapis.com" {
		p := strings.TrimPrefix(path, "/")
		parts := strings.SplitN(p, "/", 2)
		if len(parts) < 2 {
			return "", "", errors.New("invalid storage.googleapis.com url: missing /bucket/objectPath")
		}
		bucket = strings.TrimSpace(parts[0])
		objectPath = parts[1]
		return bucket, objectPath, nil
	}

	// https://<bucket>.storage.googleapis.com/object
	if strings.HasSuffix(host, ".storage.googleapis.com") {
		bucket = strings.TrimSuffix(host, ".storage.googleapis.com")
		objectPath = strings.TrimPrefix(path, "/")
		if strings.TrimSpace(bucket) == "" {
			return "", "", errors.New("invalid *.storage.googleapis.com url: bucket is empty")
		}
		return bucket, objectPath, nil
	}

	// 他ドメインの可能性があるなら、ここで許可/拒否を選べる。
	// 現状は「URL だが GCS 形式じゃない」は拒否（想定外のURLを保存しない）
	return "", "", errors.New("unsupported image url host: " + host)
}

func normalizeObjectPath(p string) string {
	s := strings.TrimSpace(p)
	if s == "" {
		return ""
	}

	// クエリが混ざっても切り落とす（念のため）
	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}

	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimPrefix(s, "/")

	// 危険な path traversal をざっくり防ぐ
	s = strings.ReplaceAll(s, "../", "")
	s = strings.ReplaceAll(s, "..\\", "")
	s = strings.ReplaceAll(s, "/..", "")

	// 連続スラッシュを潰す
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}

	return strings.TrimSpace(s)
}

// buildGCSPublicURL builds
// https://storage.googleapis.com/<bucket>/<encoded-objectPath>
//
// objectPath は "/" を残して segment 単位で PathEscape する。
func buildGCSPublicURL(bucket string, objectPath string) string {
	b := strings.TrimSpace(bucket)
	p := normalizeObjectPath(objectPath)
	if b == "" || p == "" {
		return ""
	}

	segs := strings.Split(p, "/")
	for i := range segs {
		// PathEscape は "/" を含めない前提で segment ごとに
		segs[i] = url.PathEscape(segs[i])
	}
	encoded := strings.Join(segs, "/")

	return "https://storage.googleapis.com/" + b + "/" + encoded
}

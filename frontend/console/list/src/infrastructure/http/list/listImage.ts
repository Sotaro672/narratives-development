// frontend\console\list\src\infrastructure\http\list\listImage.ts
import { asNumber } from "./number";
import { parseDateMs } from "./dates";
import { s } from "./string";
import type { ListImageDTO } from "./types";

/**
 * Firebase Storage の gs:// URL を HTTPS URL として使える形に変換する。
 *
 * ただし Firebase Storage の download token が無い場合、
 * このURLで必ず表示できるとは限らない。
 * 基本的には backend または frontend が保存した downloadURL / url / imageUrl を優先する。
 */
function buildFirebaseStorageObjectUrl(raw: string): string {
  const value = s(raw);
  if (!value) return "";

  if (value.startsWith("http://") || value.startsWith("https://")) {
    return value;
  }

  if (!value.startsWith("gs://")) {
    return "";
  }

  const withoutScheme = value.replace(/^gs:\/\//, "");
  const slashIndex = withoutScheme.indexOf("/");
  if (slashIndex <= 0) return "";

  const bucket = withoutScheme.slice(0, slashIndex);
  const objectPath = withoutScheme.slice(slashIndex + 1);

  if (!bucket || !objectPath) return "";

  return `https://firebasestorage.googleapis.com/v0/b/${encodeURIComponent(
    bucket,
  )}/o/${encodeURIComponent(objectPath)}?alt=media`;
}

/**
 * Firebase Storage の bucket + storagePath から表示用URL候補を組み立てる。
 *
 * 注意:
 * - Firebase Storage の本来の表示URLは getDownloadURL() が返す token 付きURL。
 * - ここで組み立てるURLは fallback 用。
 * - 通常は imageUrl / url / downloadURL を保存して、それを使う。
 */
function buildFirebaseStorageUrl(args: {
  bucket?: unknown;
  storageBucket?: unknown;
  storagePath?: unknown;
  path?: unknown;
}): string {
  const bucket = s(args.bucket) || s(args.storageBucket);
  const storagePath = s(args.storagePath) || s(args.path);

  if (!bucket || !storagePath) return "";

  return `https://firebasestorage.googleapis.com/v0/b/${encodeURIComponent(
    bucket,
  )}/o/${encodeURIComponent(storagePath)}?alt=media`;
}

/**
 * ✅ ListImage から "表示用URL" を解決
 *
 * 新方針:
 * - listImage は GCS ではなく Firebase Storage へ frontend から直接アップロードする
 * - 表示には getDownloadURL() で取得した URL を優先して使う
 *
 * 優先順位:
 * 1) Firebase Storage / backend が保存した完成URL
 *    - imageUrl / image_url
 *    - downloadURL / downloadUrl / download_url
 *    - publicUrl / public_url
 *    - url
 * 2) gs://... が保存されている場合は Firebase Storage URL 候補へ変換
 * 3) bucket + storagePath から Firebase Storage URL 候補を組み立て
 */
export function resolveListImageUrl(img: ListImageDTO): string {
  const directUrl =
    s((img as any)?.imageUrl) ||
    s((img as any)?.imageURL) ||
    s((img as any)?.image_url) ||
    s((img as any)?.downloadURL) ||
    s((img as any)?.downloadUrl) ||
    s((img as any)?.download_url) ||
    s((img as any)?.publicUrl) ||
    s((img as any)?.publicURL) ||
    s((img as any)?.public_url) ||
    s((img as any)?.url) ||
    s((img as any)?.URL);

  if (directUrl) return directUrl;

  const gsUrl =
    s((img as any)?.gsUrl) ||
    s((img as any)?.gsURL) ||
    s((img as any)?.gs_url) ||
    s((img as any)?.storageUrl) ||
    s((img as any)?.storageURL) ||
    s((img as any)?.storage_url);

  const firebaseUrlFromGsUrl = buildFirebaseStorageObjectUrl(gsUrl);
  if (firebaseUrlFromGsUrl) return firebaseUrlFromGsUrl;

  const storagePath =
    s((img as any)?.storagePath) ||
    s((img as any)?.storage_path) ||
    s((img as any)?.StoragePath) ||
    s((img as any)?.objectPath) ||
    s((img as any)?.object_path) ||
    s((img as any)?.ObjectPath) ||
    s((img as any)?.path) ||
    s((img as any)?.Path);

  const bucket =
    s((img as any)?.storageBucket) ||
    s((img as any)?.storage_bucket) ||
    s((img as any)?.StorageBucket) ||
    s((img as any)?.bucket) ||
    s((img as any)?.Bucket);

  const built = buildFirebaseStorageUrl({
    bucket,
    storagePath,
  });

  return built;
}

export function normalizeListImageUrls(
  listImages: ListImageDTO[],
  primaryImageId?: string,
): string[] {
  const pid = s(primaryImageId);

  const rows = (Array.isArray(listImages) ? listImages : [])
    .map((img) => {
      const id =
        s((img as any)?.id) ||
        s((img as any)?.ID) ||
        s((img as any)?.imageId) ||
        s((img as any)?.image_id);

      const url = resolveListImageUrl(img);

      const displayOrder =
        asNumber((img as any)?.displayOrder, 0) ||
        asNumber((img as any)?.display_order, 0) ||
        asNumber((img as any)?.DisplayOrder, 0) ||
        0;

      const createdAtMs =
        parseDateMs((img as any)?.createdAt) ||
        parseDateMs((img as any)?.created_at) ||
        parseDateMs((img as any)?.CreatedAt) ||
        0;

      return { id, url, displayOrder, createdAtMs };
    })
    .filter((x) => Boolean(x.url));

  rows.sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) return a.displayOrder - b.displayOrder;
    if (a.createdAtMs !== b.createdAtMs) return a.createdAtMs - b.createdAtMs;
    return a.id.localeCompare(b.id);
  });

  const out: string[] = [];
  const seen = new Set<string>();
  let primaryUrl = "";

  for (const r of rows) {
    const url = s(r.url);
    if (!url || seen.has(url)) continue;
    seen.add(url);

    if (pid && s(r.id) === pid && !primaryUrl) {
      primaryUrl = url;
      continue;
    }

    out.push(url);
  }

  if (primaryUrl) return [primaryUrl, ...out];
  return out;
}
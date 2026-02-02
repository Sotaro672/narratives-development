//frontend\console\list\src\infrastructure\http\list\listImage.ts
import { asNumber } from "./number";
import { parseDateMs } from "./dates";
import { buildPublicGcsUrl } from "./gcsUrl";
import { s } from "./string";
import type { ListImageDTO } from "./types";

/**
 * ✅ ListImage から "表示用URL" を解決
 * 優先順位:
 * 1) publicUrl/url/signedUrl 等（= backend が完成URLを返す場合）
 * 2) bucket + objectPath から public URL を組み立て（objectPathはURLエンコード）
 */
export function resolveListImageUrl(img: ListImageDTO): string {
  const u =
    s((img as any)?.publicUrl) ||
    s((img as any)?.publicURL) ||
    s((img as any)?.url) ||
    s((img as any)?.URL) ||
    s((img as any)?.signedUrl) ||
    s((img as any)?.signedURL) ||
    s((img as any)?.uploadUrl) ||
    s((img as any)?.uploadURL);

  // ✅ ここは backend が返したURLを尊重（既にエンコード済み/署名付き等の可能性）
  if (u) return u;

  const bucket = s((img as any)?.bucket) || s((img as any)?.Bucket) || "";
  const objectPath =
    s((img as any)?.objectPath) ||
    s((img as any)?.ObjectPath) ||
    s((img as any)?.path) ||
    s((img as any)?.Path) ||
    "";

  const built = buildPublicGcsUrl(bucket, objectPath);
  return built;
}

export function normalizeListImageUrls(
  listImages: ListImageDTO[],
  primaryImageId?: string,
): string[] {
  const pid = s(primaryImageId);

  const rows = (Array.isArray(listImages) ? listImages : [])
    .map((img) => {
      const id = s((img as any)?.id) || s((img as any)?.ID) || s((img as any)?.imageId);
      const url = resolveListImageUrl(img);
      const displayOrder =
        asNumber((img as any)?.displayOrder, 0) ||
        asNumber((img as any)?.DisplayOrder, 0) ||
        0;

      const createdAtMs =
        parseDateMs((img as any)?.createdAt) ||
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

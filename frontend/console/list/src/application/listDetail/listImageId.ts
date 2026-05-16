// frontend/console/list/src/application/listDetail/listImageId.ts

/**
 * Firebase Storage URL / imageId から list image document id を抽出する。
 *
 * 対応:
 * - imageId そのもの
 * - Firebase Storage download URL:
 *   .../o/lists%2F{listId}%2Fimages%2F{imageId}%2F{fileName}?alt=media&token=...
 */
export function extractListImageIdFromUrlOrObjectPath(args: {
  listId: string;
  raw: string;
}): string {
  const listId = String(args.listId ?? "").trim();
  const raw = String(args.raw ?? "").trim();

  if (!listId || !raw) return "";

  // imageId そのもの
  if (!raw.includes("/") && !raw.includes("?")) {
    return raw;
  }

  try {
    const url = new URL(raw);
    const pathname = decodeURIComponent(url.pathname);

    const marker = "/o/";
    const markerIndex = pathname.indexOf(marker);

    if (markerIndex < 0) {
      return "";
    }

    const objectPath = decodeURIComponent(pathname.slice(markerIndex + marker.length));
    const parts = objectPath.replace(/^\/+/, "").split("/").filter(Boolean);

    if (
      parts[0] === "lists" &&
      parts[1] === listId &&
      parts[2] === "images" &&
      parts[3]
    ) {
      return parts[3];
    }
  } catch {
    // noop
  }

  return "";
}
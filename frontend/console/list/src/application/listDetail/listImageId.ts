// frontend/console/list/src/application/listDetail/listImageId.ts

function toText(v: unknown): string {
  if (v === null || v === undefined) return "";
  return typeof v === "string" ? v.trim() : String(v).trim();
}

/**
 * Firebase Storage URL / objectPath / imageId から list image document id を抽出する。
 *
 * 対応:
 * - imageId そのもの
 * - lists/{listId}/images/{imageId}/{fileName}
 * - Firebase Storage download URL:
 *   .../o/lists%2F{listId}%2Fimages%2F{imageId}%2F{fileName}?alt=media&token=...
 */
export function extractListImageIdFromUrlOrObjectPath(args: {
  listId: string;
  raw: string;
}): string {
  const listId = toText(args.listId);
  const raw = toText(args.raw);

  if (!listId || !raw) return "";

  if (!raw.includes("/") && !raw.includes("?")) {
    return raw;
  }

  const tryExtractFromPath = (pathLike: string): string => {
    const path = toText(pathLike).replace(/^\/+/, "");
    if (!path) return "";

    const parts = path
      .split("/")
      .map((x) => toText(x))
      .filter(Boolean);

    const index = parts.findIndex(
      (x, idx) =>
        x === "lists" &&
        parts[idx + 1] === listId &&
        parts[idx + 2] === "images" &&
        Boolean(parts[idx + 3]),
    );

    if (index >= 0) {
      return toText(parts[index + 3]);
    }

    return "";
  };

  const direct = tryExtractFromPath(raw);
  if (direct) return direct;

  try {
    const url = new URL(raw);

    const objectPathParam = toText(url.searchParams.get("name"));
    const fromNameParam = tryExtractFromPath(objectPathParam);
    if (fromNameParam) return fromNameParam;

    const pathname = decodeURIComponent(toText(url.pathname));
    const fromPathname = tryExtractFromPath(pathname);
    if (fromPathname) return fromPathname;

    const marker = "/o/";
    const markerIndex = pathname.indexOf(marker);

    if (markerIndex >= 0) {
      const encodedObjectPath = pathname.slice(markerIndex + marker.length);
      const objectPath = decodeURIComponent(encodedObjectPath);
      const fromObjectPath = tryExtractFromPath(objectPath);
      if (fromObjectPath) return fromObjectPath;
    }
  } catch {
    // noop
  }

  return "";
}
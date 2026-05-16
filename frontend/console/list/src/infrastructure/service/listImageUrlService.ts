// frontend\console\list\src\infrastructure\service\listImageUrlService.ts

import type { ListImageDTO } from "../dto/listImageDto";

export function resolveListImageUrl(img: ListImageDTO): string {
  return img.url;
}

export function normalizeListImageUrls(
  listImages: ListImageDTO[],
  primaryImageId?: string,
): string[] {
  const pid = primaryImageId ?? "";

  const rows = (Array.isArray(listImages) ? listImages : [])
    .map((img, index) => {
      const id = img.id;
      const url = resolveListImageUrl(img);
      const displayOrderRaw = img.displayOrder;
      const displayOrder =
        displayOrderRaw === null || displayOrderRaw === undefined
          ? index
          : Number(displayOrderRaw);

      return {
        id,
        url,
        displayOrder: Number.isFinite(displayOrder) ? displayOrder : index,
        index,
      };
    })
    .filter((x) => Boolean(x.url));

  rows.sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) {
      return a.displayOrder - b.displayOrder;
    }

    return a.index - b.index;
  });

  const out: string[] = [];
  const seen = new Set<string>();
  let primaryUrl = "";

  for (const r of rows) {
    const url = r.url;
    if (!url || seen.has(url)) continue;

    seen.add(url);

    if (pid && r.id === pid && !primaryUrl) {
      primaryUrl = url;
      continue;
    }

    out.push(url);
  }

  if (primaryUrl) return [primaryUrl, ...out];
  return out;
}
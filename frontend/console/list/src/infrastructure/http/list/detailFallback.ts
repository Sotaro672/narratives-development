//frontend\console\list\src\infrastructure\http\list\detailFallback.ts
import type { ListDTO, ListImageDTO } from "./types";
import { normalizeListDocId } from "./ids";
import { s } from "./string";
import { normalizeListImageUrls } from "./listImage";

export async function ensureDetailHasImageUrls(
  dto: ListDTO,
  listIdRaw: string,
  loadImages: (listId: string) => Promise<ListImageDTO[]>,
): Promise<ListDTO> {
  const listId = normalizeListDocId(listIdRaw);
  const anyDto = dto as any;

  const currentUrls = Array.isArray(anyDto?.imageUrls) ? (anyDto.imageUrls as any[]) : [];
  const normalizedCurrent = currentUrls.map((x) => s(x)).filter(Boolean);

  if (normalizedCurrent.length > 0) {
    return {
      ...dto,
      imageUrls: normalizedCurrent,
    };
  }

  // fallback: images endpoint から生成
  try {
    const imgs = await loadImages(listId);
    const urls = normalizeListImageUrls(imgs, s(anyDto?.imageId));

    if (urls.length === 0) return dto;

    return {
      ...dto,
      imageUrls: urls,
    };
  } catch {
    // 画像取得に失敗しても detail 自体は返す（画面を壊さない）
    return dto;
  }
}

// frontend/amol/src/features/resale/presentation/utils/resaleConditionMedia.ts

import type {
  ResaleConditionMediaItem,
} from "../types/resaleCreatePageTypes";

type PreviewUrlItem = Pick<
  ResaleConditionMediaItem,
  "previewUrl"
>;

function createConditionMediaId(
  file: File,
): string {
  return [
    file.name,
    file.size,
    file.lastModified,
    crypto.randomUUID(),
  ].join("-");
}

/**
 * 再販の商品状態画像として扱えるファイルか判定する。
 */
export function isResaleConditionImageFile(
  file: File,
): boolean {
  return file.type.startsWith(
    "image/",
  );
}

/**
 * FileListまたはFile配列から画像ファイルだけを抽出する。
 */
export function filterResaleConditionImageFiles(
  files:
    | FileList
    | readonly File[]
    | null
    | undefined,
): File[] {
  return Array.from(
    files ?? [],
  ).filter(
    isResaleConditionImageFile,
  );
}

/**
 * 画像ファイルからMediaUploader表示用の項目を生成する。
 *
 * previewUrlは呼び出し側で不要になった時点で
 * revokeResaleConditionMediaPreviewを使って解放する。
 */
export function createResaleConditionMediaItem(
  file: File,
): ResaleConditionMediaItem {
  if (
    !isResaleConditionImageFile(
      file,
    )
  ) {
    throw new Error(
      "商品状態には画像ファイルのみ追加できます。",
    );
  }

  return {
    id:
      createConditionMediaId(
        file,
      ),
    type: "image",
    previewUrl:
      URL.createObjectURL(
        file,
      ),
    title: file.name,
    fileName: file.name,
    file,
  };
}

/**
 * 複数ファイルから画像項目をまとめて生成する。
 * 画像以外のファイルは除外する。
 */
export function createResaleConditionMediaItems(
  files:
    | FileList
    | readonly File[]
    | null
    | undefined,
): ResaleConditionMediaItem[] {
  return filterResaleConditionImageFiles(
    files,
  ).map(
    createResaleConditionMediaItem,
  );
}

/**
 * 1件のプレビュー用Object URLを解放する。
 */
export function revokeResaleConditionMediaPreview(
  item: PreviewUrlItem,
): void {
  if (!item.previewUrl) {
    return;
  }

  URL.revokeObjectURL(
    item.previewUrl,
  );
}

/**
 * 複数のプレビュー用Object URLをまとめて解放する。
 */
export function revokeResaleConditionMediaPreviews(
  items:
    readonly PreviewUrlItem[],
): void {
  items.forEach(
    revokeResaleConditionMediaPreview,
  );
}
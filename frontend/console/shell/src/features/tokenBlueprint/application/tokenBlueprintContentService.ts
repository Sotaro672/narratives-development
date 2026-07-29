// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintContentService.ts

import type {
  ContentFile,
  TokenBlueprint,
} from "../../../shared/types/tokenBlueprint";

import {
  TOKEN_BLUEPRINT_DEFAULT_CONTENT_TYPE,
} from "../../../shared/types/tokenBlueprint";

import {
  patchTokenBlueprintContentFiles,
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

import {
  uploadTokenBlueprintContentToFirebaseStorage,
} from "../infrastructure/storage/tokenBlueprintAssetStorage";

/**
 * TokenBlueprintコンテンツの追加処理に必要な入力値。
 */
export type UploadAndAppendTokenBlueprintContentsInput = {
  /**
   * Firebase Storage上の保存先を分離する会社ID。
   */
  companyId: string;

  /**
   * コンテンツを追加するTokenBlueprintのID。
   */
  tokenBlueprintId: string;

  /**
   * contentFilesのcreatedBy・updatedByへ保存する
   * members document ID。
   */
  actorId: string;

  /**
   * Firebase Storageへアップロードするファイル。
   */
  files: File[];

  /**
   * TokenBlueprintに現在保存されているコンテンツ。
   *
   * 新規作成直後の場合は空配列を渡す。
   */
  existingContentFiles: ContentFile[];
};

/**
 * TokenBlueprintコンテンツ用の一意なIDを生成する。
 *
 * 作成画面のローカルコンテンツIDにも利用できるよう、
 * exportしている。
 */
export function createTokenBlueprintContentId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }

  return `c_${Date.now()}_${Math.random()
    .toString(16)
    .slice(2)}`;
}

/**
 * Storageへのアップロード結果から、
 * backendへ保存するContentFileを生成する。
 */
async function uploadContentFile(params: {
  companyId: string;
  tokenBlueprintId: string;
  actorId: string;
  file: File;
}): Promise<ContentFile> {
  const contentId =
    createTokenBlueprintContentId();

  const uploaded =
    await uploadTokenBlueprintContentToFirebaseStorage({
      companyId:
        params.companyId,

      tokenBlueprintId:
        params.tokenBlueprintId,

      contentId,

      file:
        params.file,
    });

  const nowIso =
    new Date().toISOString();

  return {
    id:
      contentId,

    name:
      uploaded.fileName ||
      params.file.name ||
      contentId,

    type:
      uploaded.kind,

    contentType:
      uploaded.contentType ||
      params.file.type ||
      TOKEN_BLUEPRINT_DEFAULT_CONTENT_TYPE,

    objectPath:
      uploaded.objectPath,

    url:
      uploaded.downloadUrl,

    size:
      Number.isFinite(
        uploaded.size,
      ) &&
      uploaded.size >= 0
        ? uploaded.size
        : params.file.size,

    isPublic:
      false,

    createdAt:
      nowIso,

    createdBy:
      params.actorId,

    updatedAt:
      nowIso,

    updatedBy:
      params.actorId,
  };
}

/**
 * 既存コンテンツと新規コンテンツをID単位で結合する。
 *
 * 同じIDが存在する場合は、新しいContentFileを正とする。
 */
function mergeContentFiles(
  existingContentFiles: ContentFile[],
  newContentFiles: ContentFile[],
): ContentFile[] {
  const mergedContentFiles =
    new Map<
      string,
      ContentFile
    >();

  for (
    const contentFile of existingContentFiles
  ) {
    mergedContentFiles.set(
      contentFile.id,
      contentFile,
    );
  }

  for (
    const contentFile of newContentFiles
  ) {
    mergedContentFiles.set(
      contentFile.id,
      contentFile,
    );
  }

  return Array.from(
    mergedContentFiles.values(),
  );
}

/**
 * 選択されたファイルをFirebase Storageへアップロードし、
 * TokenBlueprintのcontentFilesへ追加する。
 *
 * 処理順:
 * 1. 入力値を検証
 * 2. ファイルを1件ずつFirebase Storageへアップロード
 * 3. アップロード結果からContentFileを生成
 * 4. 既存contentFilesと新規ContentFileを結合
 * 5. backendへcontentFilesを保存
 *
 * ファイルを1件ずつ順番にアップロードすることで、
 * 処理順とエラー発生位置を明確にする。
 */
export async function uploadAndAppendTokenBlueprintContents(
  input: UploadAndAppendTokenBlueprintContentsInput,
): Promise<TokenBlueprint> {
  if (!input.companyId) {
    throw new Error(
      "companyId is required",
    );
  }

  if (!input.tokenBlueprintId) {
    throw new Error(
      "tokenBlueprintId is required",
    );
  }

  if (!input.actorId) {
    throw new Error(
      "actorId is required",
    );
  }

  if (
    !Array.isArray(
      input.files,
    ) ||
    input.files.length === 0
  ) {
    throw new Error(
      "files must contain at least one file",
    );
  }

  if (
    !Array.isArray(
      input.existingContentFiles,
    )
  ) {
    throw new Error(
      "existingContentFiles must be an array",
    );
  }

  const newContentFiles:
    ContentFile[] = [];

  for (
    const file of input.files
  ) {
    const contentFile =
      await uploadContentFile({
        companyId:
          input.companyId,

        tokenBlueprintId:
          input.tokenBlueprintId,

        actorId:
          input.actorId,

        file,
      });

    newContentFiles.push(
      contentFile,
    );
  }

  const mergedContentFiles =
    mergeContentFiles(
      input.existingContentFiles,
      newContentFiles,
    );

  return patchTokenBlueprintContentFiles({
    tokenBlueprintId:
      input.tokenBlueprintId,

    contentFiles:
      mergedContentFiles,
  });
}
// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintContentService.ts

import type { ContentFile, TokenBlueprint } from "../../../shared/types/tokenBlueprint";
import { patchTokenBlueprintContentFiles } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { uploadTokenBlueprintContentToFirebaseStorage } from "../infrastructure/storage/tokenBlueprintAssetStorage";

export type UploadAndAppendTokenBlueprintContentsInput = {
  companyId: string;
  tokenBlueprintId: string;
  actorId: string;
  files: File[];
  existingContentFiles: ContentFile[];
};

/**
 * TokenBlueprintコンテンツ用IDを生成する。
 *
 * 作成画面のローカルコンテンツIDでも利用するためexportする。
 */
export function createTokenBlueprintContentId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }

  return `c_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

/**
 * Firebase Storageへのupload結果からBackendへ保存するContentFileを生成する。
 *
 * fileName / contentType / size / kindはStorage層の戻り値を正とし、
 * Application層ではfallback・normalizeを行わない。
 */
async function uploadContentFile(params: {
  companyId: string;
  tokenBlueprintId: string;
  actorId: string;
  file: File;
}): Promise<ContentFile> {
  const contentId = createTokenBlueprintContentId();

  const uploaded = await uploadTokenBlueprintContentToFirebaseStorage({
    companyId: params.companyId,
    tokenBlueprintId: params.tokenBlueprintId,
    contentId,
    file: params.file,
  });

  const nowIso = new Date().toISOString();

  return {
    id: contentId,
    name: uploaded.fileName,
    type: uploaded.kind,
    contentType: uploaded.contentType,
    objectPath: uploaded.objectPath,
    url: uploaded.downloadUrl,
    size: uploaded.size,
    isPublic: false,
    createdAt: nowIso,
    createdBy: params.actorId,
    updatedAt: nowIso,
    updatedBy: params.actorId,
  };
}

/**
 * 既存コンテンツと新規コンテンツをID単位で結合する。
 * 同一IDがある場合は新しいContentFileを正とする。
 */
function mergeContentFiles(
  existingContentFiles: ContentFile[],
  newContentFiles: ContentFile[],
): ContentFile[] {
  const merged = new Map<string, ContentFile>();

  for (const contentFile of existingContentFiles) {
    merged.set(contentFile.id, contentFile);
  }

  for (const contentFile of newContentFiles) {
    merged.set(contentFile.id, contentFile);
  }

  return Array.from(merged.values());
}

/**
 * 選択されたファイルをFirebase Storageへuploadし、
 * TokenBlueprint.contentFilesへ追加する。
 *
 * 処理順:
 * 1. 入力値を確認
 * 2. Firebase Storageへ順番にupload
 * 3. upload結果からContentFileを生成
 * 4. 既存contentFilesと結合
 * 5. Backendへ保存
 */
export async function uploadAndAppendTokenBlueprintContents(
  input: UploadAndAppendTokenBlueprintContentsInput,
): Promise<TokenBlueprint> {
  if (!input.companyId) {
    throw new Error("companyId is required");
  }

  if (!input.tokenBlueprintId) {
    throw new Error("tokenBlueprintId is required");
  }

  if (!input.actorId) {
    throw new Error("actorId is required");
  }

  if (input.files.length === 0) {
    throw new Error("files must contain at least one file");
  }

  const newContentFiles: ContentFile[] = [];

  for (const file of input.files) {
    const contentFile = await uploadContentFile({
      companyId: input.companyId,
      tokenBlueprintId: input.tokenBlueprintId,
      actorId: input.actorId,
      file,
    });

    newContentFiles.push(contentFile);
  }

  return patchTokenBlueprintContentFiles({
    tokenBlueprintId: input.tokenBlueprintId,
    contentFiles: mergeContentFiles(input.existingContentFiles, newContentFiles),
  });
}
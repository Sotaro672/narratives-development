// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintContentService.ts 
 
import type { ContentType } from "../../../shared/types/tokenBlueprint"; 
import { 
  registerTokenBlueprintCreateOperationContent, 
  type TokenBlueprintCreateOperation, 
} from "../infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP"; 
import { 
  uploadTokenBlueprintContentToFirebaseStorage, 
  type FirebaseStorageUploadProgress, 
} from "../infrastructure/storage/tokenBlueprintAssetStorage"; 
 
export type TokenBlueprintContentUploadInput = { 
  id: string; 
  file: File; 
  type: ContentType; 
}; 
 
export type TokenBlueprintContentUploadProgress = { 
  contentId: string; 
  fileName: string; 
  completedCount: number; 
  totalCount: number; 
  transferredBytes: number; 
  totalBytes: number; 
  percentage: number; 
}; 
 
export type UploadTokenBlueprintCreateOperationContentsInput = { 
  companyId: string; 
  tokenBlueprintId: string; 
  operationId: string; 
  contents: TokenBlueprintContentUploadInput[]; 
  onProgress?: (progress: TokenBlueprintContentUploadProgress) => void; 
}; 
 
export type UploadTokenBlueprintCreateOperationContentsResult = { 
  operation: TokenBlueprintCreateOperation; 
  uploadedContentIds: string[]; 
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
 
function requireContentUploadInput( 
  input: TokenBlueprintContentUploadInput, 
): void { 
  if (!input.id) { 
    throw new Error("content.id is required"); 
  } 
 
  if (!input.file) { 
    throw new Error("content.file is required"); 
  } 
 
  if (!input.file.name) { 
    throw new Error("content.file.name is required"); 
  } 
 
  if (!input.type) { 
    throw new Error("content.type is required"); 
  } 
} 
 
function calculatePercentage( 
  transferredBytes: number, 
  totalBytes: number, 
): number { 
  if (totalBytes <= 0) { 
    return 100; 
  } 
 
  return Math.min( 
    100, 
    Math.max( 
      0, 
      Math.round( 
        (transferredBytes / totalBytes) * 100, 
      ), 
    ), 
  ); 
} 
 
function calculateTotalBytes( 
  contents: TokenBlueprintContentUploadInput[], 
): number { 
  return contents.reduce( 
    (total, content) => total + content.file.size, 
    0, 
  ); 
} 
 
/** 
 * 選択されたコンテンツをFirebase Storageへ順番にuploadし、 
 * upload完了ごとにCreate Operationへ即時登録する。 
 * 
 * TokenBlueprint.contentFilesはこの処理では直接更新しない。 
 * 最終的なContentFileへの反映はCloud Tasks workerが行う。 
 * 
 * 処理順: 
 * 1. 入力値を確認 
 * 2. Firebase Storageへ1件upload 
 * 3. upload完了直後にCreate Operationへ登録 
 * 4. 次のファイルをupload 
 * 5. 全件登録済みのOperationを返す 
 */ 
export async function uploadTokenBlueprintCreateOperationContents( 
  input: UploadTokenBlueprintCreateOperationContentsInput, 
): Promise<UploadTokenBlueprintCreateOperationContentsResult> { 
  if (!input.companyId) { 
    throw new Error("companyId is required"); 
  } 
 
  if (!input.tokenBlueprintId) { 
    throw new Error("tokenBlueprintId is required"); 
  } 
 
  if (!input.operationId) { 
    throw new Error("operationId is required"); 
  } 
 
  if (input.contents.length === 0) { 
    throw new Error("contents must contain at least one content"); 
  } 
 
  for (const content of input.contents) { 
    requireContentUploadInput(content); 
  } 
 
  const totalBytes = calculateTotalBytes(input.contents); 
 
  const transferredByContentId = new Map<string, number>(); 
 
  const uploadedContentIds: string[] = []; 
 
  let completedCount = 0; 
 
  let latestOperation: TokenBlueprintCreateOperation | null = null; 
 
  const emitProgress = ( 
    content: TokenBlueprintContentUploadInput, 
    progress: FirebaseStorageUploadProgress, 
  ): void => { 
    transferredByContentId.set( 
      content.id, 
      progress.transferredBytes, 
    ); 
 
    let transferredBytes = 0; 
 
    for (const value of transferredByContentId.values()) { 
      transferredBytes += value; 
    } 
 
    input.onProgress?.({ 
      contentId: content.id, 
      fileName: content.file.name, 
      completedCount, 
      totalCount: input.contents.length, 
      transferredBytes, 
      totalBytes, 
      percentage: calculatePercentage( 
        transferredBytes, 
        totalBytes, 
      ), 
    }); 
  }; 
 
  for (const content of input.contents) { 
    transferredByContentId.set( 
      content.id, 
      0, 
    ); 
 
    const uploaded = await uploadTokenBlueprintContentToFirebaseStorage({ 
      companyId: input.companyId, 
      tokenBlueprintId: input.tokenBlueprintId, 
      contentId: content.id, 
      file: content.file, 
      onProgress: (progress) => { 
        emitProgress( 
          content, 
          progress, 
        ); 
      }, 
    }); 
 
    if (uploaded.kind !== content.type) { 
      throw new Error( 
        `content type mismatch: contentId=${content.id}, expected=${content.type}, actual=${uploaded.kind}`, 
      ); 
    } 
 
    latestOperation = await registerTokenBlueprintCreateOperationContent( 
      input.operationId, 
      content.id, 
      { 
        url: uploaded.downloadUrl, 
        objectPath: uploaded.objectPath, 
        name: uploaded.fileName, 
        contentType: uploaded.contentType, 
        size: uploaded.size, 
      }, 
    ); 
 
    transferredByContentId.set( 
      content.id, 
      uploaded.size, 
    ); 
 
    uploadedContentIds.push( 
      content.id, 
    ); 
 
    completedCount += 1; 
 
    let transferredBytes = 0; 
 
    for (const value of transferredByContentId.values()) { 
      transferredBytes += value; 
    } 
 
    input.onProgress?.({ 
      contentId: content.id, 
      fileName: content.file.name, 
      completedCount, 
      totalCount: input.contents.length, 
      transferredBytes, 
      totalBytes, 
      percentage: calculatePercentage( 
        transferredBytes, 
        totalBytes, 
      ), 
    }); 
  } 
 
  if (!latestOperation) { 
    throw new Error("token blueprint create operation was not updated"); 
  } 
 
  return { 
    operation: latestOperation, 
    uploadedContentIds, 
  }; 
} 
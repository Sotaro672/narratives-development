// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintCreateService.tsx 
 
import { fetchBrandsForCurrentCompany } from "../../brand/infrastructure/http/brandRepositoryHTTP"; 
import { 
  commitTokenBlueprintCreateOperation, 
  registerTokenBlueprintCreateOperationIcon, 
  startTokenBlueprintCreateOperation, 
  type TokenBlueprintCreateOperation, 
} from "../infrastructure/repository/tokenBlueprintCreateOperationRepositoryHTTP"; 
import { 
  fetchTokenBlueprintById, 
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP"; 
import { 
  getTokenBlueprintContentType, 
  uploadTokenBlueprintIconToFirebaseStorage, 
  type FirebaseStorageUploadProgress, 
} from "../infrastructure/storage/tokenBlueprintAssetStorage"; 
import { 
  uploadTokenBlueprintCreateOperationContents, 
  type TokenBlueprintContentUploadInput, 
  type TokenBlueprintContentUploadProgress, 
} from "./tokenBlueprintContentService"; 
 
export type TokenBlueprintBrandOption = { 
  id: string; 
  name: string; 
}; 
 
/** 
 * TokenBlueprintCardで選択可能なブランドを取得する。 
 * 
 * /brandsのresponseを正とし、Frontend側でfallbackや個別名前解決は行わない。 
 */ 
export async function loadBrandsForCompany(): Promise<TokenBlueprintBrandOption[]> { 
  return fetchBrandsForCurrentCompany(); 
} 
 
/** 
 * TokenBlueprint Create Operation用のidempotency keyを生成する。 
 * 
 * Hook側でOperation開始前に生成し、sessionStorageへ保存してから 
 * createTokenBlueprintWithOptionalIconへ渡す。 
 * 
 * Start requestが通信断等で結果不明になった場合も、 
 * 同じidempotency keyを使って再送できるようにする。 
 */ 
export function createTokenBlueprintCreateOperationIdempotencyKey(): string { 
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") { 
    return `tbco_create_${crypto.randomUUID()}`; 
  } 
 
  return `tbco_create_${Date.now()}_${Math.random().toString(16).slice(2)}`; 
} 
 
export type CreateTokenBlueprintInput = { 
  name: string; 
  symbol: string; 
  brandId: string; 
  description: string; 
  assigneeId: string; 
  idempotencyKey: string; 
  operationId?: string; 
  iconFile?: File | null; 
  contents?: TokenBlueprintContentUploadInput[]; 
  maxRetries?: number; 
  onOperationChange?: (operation: TokenBlueprintCreateOperation) => void; 
  onIconProgress?: (progress: FirebaseStorageUploadProgress) => void; 
  onContentProgress?: (progress: TokenBlueprintContentUploadProgress) => void; 
}; 
 
function requireCreateInput( 
  input: CreateTokenBlueprintInput, 
): void { 
  if (!input.name) { 
    throw new Error("name is required"); 
  } 
 
  if (!input.symbol) { 
    throw new Error("symbol is required"); 
  } 
 
  if (!input.brandId) { 
    throw new Error("brandId is required"); 
  } 
 
  if (!input.assigneeId) { 
    throw new Error("assigneeId is required"); 
  } 
 
  if (!input.idempotencyKey.trim()) { 
    throw new Error("idempotencyKey is required"); 
  } 
 
  for (const content of input.contents ?? []) { 
    if (!content.id) { 
      throw new Error("content.id is required"); 
    } 
 
    if (!content.file) { 
      throw new Error("content.file is required"); 
    } 
 
    if (!content.file.name) { 
      throw new Error("content.file.name is required"); 
    } 
 
    if (!content.type) { 
      throw new Error("content.type is required"); 
    } 
  } 
} 
 
function assertOperationPlanMatchesInput( 
  operation: TokenBlueprintCreateOperation, 
  input: CreateTokenBlueprintInput, 
): void { 
  const iconFile = input.iconFile ?? null; 
  const contents = input.contents ?? []; 
 
  if (Boolean(operation.icon) !== Boolean(iconFile)) { 
    throw new Error( 
      "token blueprint create operation icon plan does not match current input", 
    ); 
  } 
 
  if (operation.icon && iconFile) { 
    const contentType = getTokenBlueprintContentType(iconFile); 
 
    if ( 
      operation.icon.fileName !== iconFile.name || 
      operation.icon.contentType !== contentType || 
      operation.icon.size !== iconFile.size 
    ) { 
      throw new Error( 
        "token blueprint create operation icon metadata does not match current file", 
      ); 
    } 
  } 
 
  if (operation.contents.length !== contents.length) { 
    throw new Error( 
      "token blueprint create operation content plan does not match current input", 
    ); 
  } 
 
  const currentContentsById = new Map( 
    contents.map((content) => [ 
      content.id, 
      content, 
    ]), 
  ); 
 
  for (const expectedContent of operation.contents) { 
    const content = currentContentsById.get(expectedContent.id); 
 
    if (!content) { 
      throw new Error( 
        `content file is required to resume operation: ${expectedContent.id}`, 
      ); 
    } 
 
    const contentType = getTokenBlueprintContentType(content.file); 
 
    if ( 
      expectedContent.name !== content.file.name || 
      expectedContent.type !== content.type || 
      expectedContent.contentType !== contentType || 
      expectedContent.size !== content.file.size 
    ) { 
      throw new Error( 
        `token blueprint create operation content metadata does not match current file: ${expectedContent.id}`, 
      ); 
    } 
  } 
} 
 
function isBrowserUploadRequired( 
  operation: TokenBlueprintCreateOperation, 
): boolean { 
  if (operation.icon && !operation.icon.uploaded) { 
    return true; 
  } 
 
  return operation.contents.some( 
    (content) => !content.uploaded, 
  ); 
} 
 
/** 
 * TokenBlueprintをCreate Operation経由で作成する。 
 * 
 * 処理順: 
 * 1. Create Operationを開始し、Backend側でTokenBlueprint本体を作成 
 * 2. Operation IDを即時Frontendへ通知 
 * 3. TokenBlueprint本体を取得してcompanyIdを確定 
 * 4. iconをFirebase Storageへupload 
 * 5. icon upload結果をCreate Operationへ即時登録 
 * 6. contentsを1件ずつFirebase Storageへupload 
 * 7. 各content upload完了直後にCreate Operationへ即時登録 
 * 8. 全upload完了後にcommit 
 * 9. Cloud Tasksへ引き継がれたOperationを返す 
 * 
 * queued以降はブラウザに依存しない。 
 * Cloud Tasks workerがTokenBlueprintのicon/contentFilesを確定する。 
 */ 
export async function createTokenBlueprintWithOptionalIcon( 
  input: CreateTokenBlueprintInput, 
): Promise<TokenBlueprintCreateOperation> { 
  requireCreateInput(input); 
 
  const iconFile = input.iconFile ?? null; 
  const contents = input.contents ?? []; 
 
  let operation = await startTokenBlueprintCreateOperation({ 
    operationId: input.operationId, 
    idempotencyKey: input.idempotencyKey, 
    name: input.name, 
    symbol: input.symbol, 
    brandId: input.brandId, 
    description: input.description, 
    assigneeId: input.assigneeId, 
    icon: iconFile 
      ? { 
          fileName: iconFile.name, 
          contentType: getTokenBlueprintContentType(iconFile), 
          size: iconFile.size, 
        } 
      : undefined, 
    contents: contents.map((content) => ({ 
      id: content.id, 
      name: content.file.name, 
      type: content.type, 
      contentType: getTokenBlueprintContentType(content.file), 
      size: content.file.size, 
    })), 
    maxRetries: input.maxRetries, 
  }); 
 
  input.onOperationChange?.(operation); 
 
  if (operation.status !== "waiting_upload") { 
    return operation; 
  } 
 
  assertOperationPlanMatchesInput( 
    operation, 
    input, 
  ); 
 
  if (!isBrowserUploadRequired(operation)) { 
    operation = await commitTokenBlueprintCreateOperation( 
      operation.id, 
    ); 
 
    input.onOperationChange?.(operation); 
 
    return operation; 
  } 
 
  const tokenBlueprint = await fetchTokenBlueprintById( 
    operation.tokenBlueprintId, 
  ); 
 
  if (!tokenBlueprint.companyId) { 
    throw new Error("token blueprint companyId is required"); 
  } 
 
  if (operation.icon && !operation.icon.uploaded) { 
    if (!iconFile) { 
      throw new Error( 
        "icon file is required to resume token blueprint create operation", 
      ); 
    } 
 
    const uploadedIcon = await uploadTokenBlueprintIconToFirebaseStorage({ 
      companyId: tokenBlueprint.companyId, 
      tokenBlueprintId: operation.tokenBlueprintId, 
      file: iconFile, 
      onProgress: input.onIconProgress, 
    }); 
 
    operation = await registerTokenBlueprintCreateOperationIcon( 
      operation.id, 
      { 
        url: uploadedIcon.downloadUrl, 
        objectPath: uploadedIcon.objectPath, 
        fileName: uploadedIcon.fileName, 
        contentType: uploadedIcon.contentType, 
        size: uploadedIcon.size, 
      }, 
    ); 
 
    input.onOperationChange?.(operation); 
  } 
 
  const uploadedContentIds = new Set( 
    operation.contents 
      .filter((content) => content.uploaded) 
      .map((content) => content.id), 
  ); 
 
  const pendingContents = contents.filter( 
    (content) => !uploadedContentIds.has(content.id), 
  ); 
 
  if (pendingContents.length > 0) { 
    const contentUploadResult = 
      await uploadTokenBlueprintCreateOperationContents({ 
        companyId: tokenBlueprint.companyId, 
        tokenBlueprintId: operation.tokenBlueprintId, 
        operationId: operation.id, 
        contents: pendingContents, 
        onProgress: input.onContentProgress, 
      }); 
 
    operation = contentUploadResult.operation; 
 
    input.onOperationChange?.(operation); 
  } 
 
  operation = await commitTokenBlueprintCreateOperation( 
    operation.id, 
  ); 
 
  input.onOperationChange?.(operation); 
 
  return operation; 
} 
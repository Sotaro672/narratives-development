// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintDetailService.tsx 
 
import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint"; 
import type { UpdateTokenBlueprintPayload } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP"; 
import { 
  deleteTokenBlueprint, 
  fetchTokenBlueprintById, 
  updateTokenBlueprint, 
} from "../infrastructure/repository/tokenBlueprintRepositoryHTTP"; 
import { 
  uploadTokenBlueprintIconToFirebaseStorage, 
  type FirebaseStorageUploadProgressHandler, 
} from "../infrastructure/storage/tokenBlueprintAssetStorage"; 
 
type TokenBlueprintCardUpdateInput = { 
  name: string; 
  symbol: string; 
  description: string; 
  iconFile?: File | null; 
}; 
 
export type UpdateTokenBlueprintProgressHandlers = { 
  onIconProgress?: FirebaseStorageUploadProgressHandler; 
  onSaving?: () => void; 
}; 
 
/** 
 * 詳細取得。 
 * 
 * Backend BFFのresponseをそのままTokenBlueprintとして扱う。 
 */ 
export async function fetchTokenBlueprintDetail(id: string): Promise<TokenBlueprint> { 
  if (!id) { 
    throw new Error("id is required"); 
  } 
 
  return fetchTokenBlueprintById(id); 
} 
 
/** 
 * TokenBlueprintを物理削除する。 
 * 
 * backend側で以下を実行する。 
 * - minted=trueの場合は削除不可 
 * - Firebase Storageのtoken-blueprints/{companyId}/{tokenBlueprintId}/prefix以下を全削除 
 * - tokenBlueprintReviews/{tokenBlueprintId}を物理削除 
 * - token_blueprints/{tokenBlueprintId}を物理削除 
 */ 
export async function deleteTokenBlueprintDetail(id: string): Promise<void> { 
  if (!id) { 
    throw new Error("id is required"); 
  } 
 
  await deleteTokenBlueprint(id); 
} 
 
/** 
 * TokenBlueprintCardの現在値から通常更新payloadを組み立てる。 
 * 
 * Backend BFF / Update API contractを正とし、Frontend側で 
 * normalize・fallback・ContentFile再mapperは行わない。 
 * 
 * contentFilesは専用のcontent更新処理で扱う。 
 * icon情報は新しいiconFileが選択された場合のみ、 
 * Firebase Storage upload後の確定値を別PUTする。 
 */ 
export function buildUpdatePayloadFromCardVm( 
  blueprint: TokenBlueprint, 
  cardVm: TokenBlueprintCardUpdateInput, 
): UpdateTokenBlueprintPayload { 
  return { 
    name: cardVm.name, 
    symbol: cardVm.symbol, 
    description: cardVm.description, 
    assigneeId: blueprint.assigneeId, 
  }; 
} 
 
/** 
 * TokenBlueprintCardの現在値からTokenBlueprintを更新する。 
 * 
 * iconFileなし: 
 * 1. 保存開始を通知 
 * 2. name / symbol / description / assigneeIdを更新 
 * 3. Backend BFFの更新responseをそのまま返す 
 * 
 * iconFileあり: 
 * 1. 保存開始を通知 
 * 2. 通常項目を更新 
 * 3. Firebase StorageへiconFileをuploadし、進捗を通知 
 * 4. 再度保存開始を通知 
 * 5. upload済みの確定icon情報を更新 
 * 6. Backend BFFの更新responseをそのまま返す 
 */ 
export async function updateTokenBlueprintFromCard( 
  blueprint: TokenBlueprint, 
  cardVm: TokenBlueprintCardUpdateInput, 
  progressHandlers?: UpdateTokenBlueprintProgressHandlers, 
): Promise<TokenBlueprint> { 
  progressHandlers?.onSaving?.(); 
 
  const updated = await updateTokenBlueprint( 
    blueprint.id, 
    buildUpdatePayloadFromCardVm(blueprint, cardVm), 
  ); 
 
  const iconFile = cardVm.iconFile ?? null; 
 
  if (!iconFile) { 
    return updated; 
  } 
 
  const uploaded = await uploadTokenBlueprintIconToFirebaseStorage({ 
    companyId: updated.companyId, 
    tokenBlueprintId: updated.id, 
    file: iconFile, 
    onProgress: progressHandlers?.onIconProgress, 
  }); 
 
  progressHandlers?.onSaving?.(); 
 
  return updateTokenBlueprint(updated.id, { 
    iconUrl: uploaded.downloadUrl, 
    iconObjectPath: uploaded.objectPath, 
    iconFileName: uploaded.fileName, 
    iconContentType: uploaded.contentType, 
    iconSize: uploaded.size, 
  }); 
}
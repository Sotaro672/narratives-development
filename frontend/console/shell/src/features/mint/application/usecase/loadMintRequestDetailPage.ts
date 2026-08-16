// frontend/console/shell/src/features/mint/application/usecase/loadMintRequestDetailPage.ts

import type { MintRequestManagementRowDTO } from "../../infrastructure/dto/mintRequestManagementRow";
import type {
  MintProductBlueprintDTO,
  MintRequestDetailDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";
import { fetchMintRequestDetailHTTP } from "../../infrastructure/repository/http/inspections";
import { fetchMintRequestRowByProductionIdHTTP } from "../../infrastructure/repository/http/mintRequests";
import { fetchMintProductBlueprintHTTP } from "../../infrastructure/repository/http/mintProductBlueprint";

export type LoadMintRequestDetailPageResult = {
  detail: MintRequestDetailDTO | null;
  row: MintRequestManagementRowDTO | null;
  productBlueprint: MintProductBlueprintDTO | null;
};

/**
 * Mint申請詳細画面の初期表示に必要なBackend BFF responseを取得する。
 *
 * Backend responseをそのまま正とし、Frontend独自DTOへの再構築やfallbackは行わない。
 *
 * 取得順序:
 * 1. inspection detail と mint request row を並列取得
 * 2. detail.productBlueprintId が存在する場合のみ ProductBlueprint を取得
 */
export async function loadMintRequestDetailPage(
  productionId: string,
): Promise<LoadMintRequestDetailPageResult> {
  if (!productionId) {
    throw new Error("productionId が空です");
  }

  const [detail, row] = await Promise.all([
    fetchMintRequestDetailHTTP(productionId),
    fetchMintRequestRowByProductionIdHTTP(productionId),
  ]);

  const productBlueprint = detail?.productBlueprintId
    ? await fetchMintProductBlueprintHTTP(detail.productBlueprintId)
    : null;

  return {
    detail,
    row,
    productBlueprint,
  };
}
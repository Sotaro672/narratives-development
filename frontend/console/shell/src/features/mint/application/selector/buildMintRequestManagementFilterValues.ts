// frontend/console/shell/src/features/mintRequest/application/selector/buildMintRequestManagementFilterValues.ts

import type {
  ViewRow as MintRequestManagementRow,
} from "../usecase/loadMintRequestManagementRows";

export type MintRequestManagementFilterValues = {
  tokenNames: string[];
  productNames: string[];
  requesterNames: string[];

  inspectionStatuses:
    MintRequestManagementRow["inspectionStatus"][];
};

function addTextIfPresent(
  values: Set<string>,
  value:
    | string
    | null
    | undefined,
): void {
  if (!value) {
    return;
  }

  values.add(value);
}

/**
 * Mint申請一覧からフィルター候補となる
 * ユニーク値を抽出する。
 *
 * 表示用labelの生成はPresentation層で行う。
 */
export function buildMintRequestManagementFilterValues(
  rows:
    readonly MintRequestManagementRow[],
): MintRequestManagementFilterValues {
  const tokenNames = new Set<string>();
  const productNames =new Set<string>();
  const requesterNames = new Set<string>();
  const inspectionStatuses = new Set<MintRequestManagementRow["inspectionStatus"]>();

  for (const row of rows) {
    addTextIfPresent(
      tokenNames,
      row.tokenName,
    );

    addTextIfPresent(
      productNames,
      row.productName,
    );

    addTextIfPresent(
      requesterNames,
      row.requestedByName,
    );

    if (row.inspectionStatus) {
      inspectionStatuses.add(
        row.inspectionStatus,
      );
    }
  }

  return {
    tokenNames:Array.from(tokenNames,),
    productNames:Array.from(productNames,),
    requesterNames:Array.from(requesterNames,),
    inspectionStatuses:Array.from(inspectionStatuses,),
  };
}
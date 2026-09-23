// frontend/console/shell/src/features/mintRequest/application/selector/buildMintRequestManagementFilterValues.ts

import type {
  ViewRow as MintRequestManagementRow,
} from "../usecase/loadMintRequestManagementRows";

export type MintRequestManagementFilterStatus =
  | "minted"
  | "minting"
  | MintRequestManagementRow["inspectionStatus"];

export type MintRequestManagementFilterValues = {
  tokenNames: string[];
  productNames: string[];
  requesterNames: string[];
  statuses: MintRequestManagementFilterStatus[];
};

function addTextIfPresent(
  values: Set<string>,
  value: string | null | undefined,
): void {
  if (!value) {
    return;
  }

  values.add(value);
}

/**
 * 一覧画面上で使用するステータスを返す。
 *
 * Mint処理中・完了の場合はMint状態を優先し、
 * それ以前は検査ステータスを使用する。
 */
export function getMintRequestManagementFilterStatus(
  row: MintRequestManagementRow,
): MintRequestManagementFilterStatus {
  if (row.status === "minted") {
    return "minted";
  }

  if (row.status === "minting") {
    return "minting";
  }

  return row.inspectionStatus;
}

/**
 * Mint申請一覧からフィルター候補となる
 * ユニーク値を抽出する。
 *
 * ステータスは一覧画面上に実際に表示される値を使用する。
 * 表示用labelの生成はPresentation層で行う。
 */
export function buildMintRequestManagementFilterValues(
  rows: readonly MintRequestManagementRow[],
): MintRequestManagementFilterValues {
  const tokenNames = new Set<string>();
  const productNames = new Set<string>();
  const requesterNames = new Set<string>();
  const statuses = new Set<MintRequestManagementFilterStatus>();

  for (const row of rows) {
    addTextIfPresent(tokenNames, row.tokenName);
    addTextIfPresent(productNames, row.productName);
    addTextIfPresent(requesterNames, row.requestedByName);

    statuses.add(
      getMintRequestManagementFilterStatus(row),
    );
  }

  return {
    tokenNames: Array.from(tokenNames),
    productNames: Array.from(productNames),
    requesterNames: Array.from(requesterNames),
    statuses: Array.from(statuses),
  };
}
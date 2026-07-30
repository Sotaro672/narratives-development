// frontend/console/shell/src/features/mintRequest/application/selector/selectMintRequestDetailState.ts

import type { MintInfo } from "../mapper/mintInfoMapper";

export type SelectMintRequestDetailStateInput = {
  /**
   * MintDTOからApplication層で変換済みのMint情報。
   */
  mint: MintInfo | null;

  /**
   * MintにbrandIdがない場合の補完値。
   *
   * ProductBlueprintPatchDTO全体には依存せず、
   * Application層に必要なbrandIdだけを受け取る。
   */
  productBlueprintBrandId?:
    | string
    | null;
};

export type MintRequestDetailState = {
  mint: MintInfo | null;

  hasMint: boolean;
  isMinting: boolean;
  isMintCompleted: boolean;

  /**
   * mintsドキュメントを作成した人の表示値。
   *
   * 優先順位:
   * 1. createdByName
   * 2. createdBy
   */
  createdByName: string | null;

  /**
   * Mint申請ボタンを押した人の表示値。
   *
   * 優先順位:
   * 1. requestedByName
   * 2. requestedBy
   */
  requestedByName: string | null;

  mintRequestedTokenBlueprintId: string;
  mintRequestedBrandId: string;
};

/**
 * Mint情報から詳細画面で使用する状態を算出する。
 *
 * Reactには依存しない純粋関数とし、
 * Presentation層ではuseMemoの中から呼び出す。
 */
export function selectMintRequestDetailState(
  input: SelectMintRequestDetailStateInput,
): MintRequestDetailState {
  const {
    mint,
    productBlueprintBrandId,
  } = input;

  const hasMint =
    mint !== null;

  /**
   * Mintが存在し、Backend上でMINTEDになる前は
   * 処理中として扱う。
   */
  const isMinting =
    hasMint &&
    mint.status !== "MINTED";

  const isMintCompleted =
    mint?.status === "MINTED";

  /**
   * createdBy系とrequestedBy系は
   * 相互に補完しない。
   */
  const createdByName =
    mint?.createdByName ||
    mint?.createdBy ||
    null;

  const requestedByName =
    mint?.requestedByName ||
    mint?.requestedBy ||
    null;

  const mintRequestedTokenBlueprintId =
    mint?.tokenBlueprintId ||
    "";

  const mintRequestedBrandId =
    mint?.brandId ||
    productBlueprintBrandId ||
    "";

  return {
    mint,

    hasMint,
    isMinting,
    isMintCompleted,

    createdByName,
    requestedByName,

    mintRequestedTokenBlueprintId,
    mintRequestedBrandId,
  };
}
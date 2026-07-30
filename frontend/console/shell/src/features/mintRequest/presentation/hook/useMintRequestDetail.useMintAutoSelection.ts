// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestDetail.useMintAutoSelection.ts

import * as React from "react";

export type UseMintAutoSelectionParams = {
  hasMint: boolean;

  mintRequestedBrandId: string;
  selectedBrandId: string;

  handleSelectBrand: (
    brandId: string,
  ) => Promise<void> | void;

  mintRequestedTokenBlueprintId: string;
  selectedTokenBlueprintId: string;

  setSelectedTokenBlueprintId:
    React.Dispatch<
      React.SetStateAction<string>
    >;

  mintScheduledBurnDate:
    | string
    | null
    | undefined;

  scheduledBurnDate: string;

  setScheduledBurnDate:
    React.Dispatch<
      React.SetStateAction<string>
    >;
};

export function useMintAutoSelection({
  hasMint,

  mintRequestedBrandId,
  selectedBrandId,
  handleSelectBrand,

  mintRequestedTokenBlueprintId,
  selectedTokenBlueprintId,
  setSelectedTokenBlueprintId,

  mintScheduledBurnDate,
  scheduledBurnDate,
  setScheduledBurnDate,
}: UseMintAutoSelectionParams): void {
  /**
   * MintにブランドIDが設定されており、
   * 画面上でブランドがまだ選択されていない場合に
   * Mintのブランドを初期選択する。
   */
  React.useEffect(() => {
    if (!hasMint) {
      return;
    }

    if (!mintRequestedBrandId) {
      return;
    }

    if (selectedBrandId) {
      return;
    }

    void Promise.resolve(
      handleSelectBrand(
        mintRequestedBrandId,
      ),
    ).catch(() => {
      // ブランド取得失敗は呼び出し側で処理する。
    });
  }, [
    hasMint,
    mintRequestedBrandId,
    selectedBrandId,
    handleSelectBrand,
  ]);

  /**
   * MintにToken Blueprint IDが設定されており、
   * 画面上でまだ選択されていない場合に初期選択する。
   */
  React.useEffect(() => {
    if (!hasMint) {
      return;
    }

    if (
      !mintRequestedTokenBlueprintId
    ) {
      return;
    }

    if (selectedTokenBlueprintId) {
      return;
    }

    setSelectedTokenBlueprintId(
      mintRequestedTokenBlueprintId,
    );
  }, [
    hasMint,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
  ]);

  /**
   * Mintに焼却予定日が設定されており、
   * 入力欄が空の場合に日付部分だけを初期反映する。
   */
  React.useEffect(() => {
    if (!hasMint) {
      return;
    }

    if (scheduledBurnDate) {
      return;
    }

    if (!mintScheduledBurnDate) {
      return;
    }

    const dateInputValue =
      mintScheduledBurnDate.length >= 10
        ? mintScheduledBurnDate.slice(
            0,
            10,
          )
        : mintScheduledBurnDate;

    if (!dateInputValue) {
      return;
    }

    setScheduledBurnDate(
      dateInputValue,
    );
  }, [
    hasMint,
    mintScheduledBurnDate,
    scheduledBurnDate,
    setScheduledBurnDate,
  ]);
}
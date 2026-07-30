// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenBlueprintCard.tsx

import * as React from "react";
import { Link2, Upload, X } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";
import { Badge } from "../../../../shared/ui/badge";
import { Label } from "../../../../shared/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shared/ui/popover";

export type TokenBlueprintCardViewModel = {
  id: string;
  name: string;
  symbol: string;
  brandId: string;
  brandName: string;
  description: string;
  iconUrl?: string;

  // mint済みの場合に、トークン名・シンボル・ブランドを
  // 編集不可にするための判定値。
  minted: boolean;

  // UIで選択されたアイコンファイル。
  iconFile?: File | null;

  // UI state
  isEditMode: boolean;
  brandOptions: {
    id: string;
    name: string;
  }[];
};

export type TokenBlueprintCardHandlers = {
  onChangeName?: (value: string) => void;
  onChangeSymbol?: (value: string) => void;
  onChangeBrand?: (
    id: string,
    name: string,
  ) => void;
  onChangeDescription?: (
    value: string,
  ) => void;

  descriptionRef?: React.RefObject<HTMLTextAreaElement>;
  iconInputRef?: React.RefObject<HTMLInputElement>;
  onRequestPickIconFile?: () => void;
  onIconInputChange?: (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => void;

  onClearLocalIconFile?: () => void;
  onToggleEditMode?: () => void;

  setEditMode?: (edit: boolean) => void;
  reset?: () => void;
};

export default function TokenBlueprintCard({
  vm,
  handlers = {},
}: {
  vm: TokenBlueprintCardViewModel;
  handlers?: TokenBlueprintCardHandlers;
}) {
  /**
   * tokenIconの選択・アップロード操作は、
   * mintedの状態にかかわらずeditモードでのみ許可する。
   */
  const canEditIcon =
    vm.isEditMode;

  /**
   * mint済みトークンでは、editモードへ移行しても
   * トークン名・シンボル・ブランドを変更不可にする。
   */
  const isIdentityLocked = Boolean(
    vm.isEditMode &&
      vm.minted,
  );

  const selectedIconFile =
    vm.iconFile ?? null;

  return (
    <Card className="token-blueprint-card">
      <CardHeader className="token-blueprint-card__header">
        <div className="token-blueprint-card__header-left">
          <span className="token-blueprint-card__header-icon">
            <Link2 className="token-blueprint-card__link-icon" />
          </span>

          <CardTitle className="token-blueprint-card__header-title">
            {vm.id
              ? "トークン設計"
              : "トークン：新規トークン設計"}
          </CardTitle>

          <Badge className="token-blueprint-card__header-badge">
            設計情報
          </Badge>
        </div>
      </CardHeader>

      <CardContent>
        <div className="token-blueprint-card__top">
          <div className="token-blueprint-card__icon-area">
            <div className="token-blueprint-card__icon-wrap">
              {vm.iconUrl ? (
                <img
                  src={vm.iconUrl}
                  alt="Token Icon"
                  className={`token-blueprint-card__icon-image${
                    canEditIcon
                      ? " is-clickable"
                      : ""
                  }`}
                  onClick={
                    canEditIcon
                      ? () => {
                          handlers.onRequestPickIconFile?.();
                        }
                      : undefined
                  }
                />
              ) : canEditIcon ? (
                <button
                  type="button"
                  className={[
                    "token-blueprint-card__icon-placeholder",
                    "token-blueprint-card__icon-placeholder-button",
                  ].join(" ")}
                  onClick={() => {
                    handlers.onRequestPickIconFile?.();
                  }}
                  aria-label="アイコン画像をアップロード"
                >
                  アイコン画像を
                  <br />
                  アップロード
                </button>
              ) : (
                <div
                  className={[
                    "token-blueprint-card__icon-placeholder",
                    "token-blueprint-card__icon-placeholder-view",
                  ].join(" ")}
                >
                  アイコン未設定
                </div>
              )}
            </div>

            {canEditIcon && (
              <>
                <input
                  ref={
                    handlers.iconInputRef ??
                    undefined
                  }
                  type="file"
                  accept="image/*"
                  className="token-blueprint-card__icon-input"
                  onChange={
                    handlers.onIconInputChange
                  }
                />

                <button
                  type="button"
                  className="token-blueprint-card__upload-btn"
                  onClick={() => {
                    handlers.onRequestPickIconFile?.();
                  }}
                >
                  <Upload className="token-blueprint-card__upload-icon" />
                  アップロード
                </button>
              </>
            )}

            {canEditIcon &&
              selectedIconFile && (
                <div className="token-blueprint-card__icon-selected">
                  <span>
                    選択中：
                    {selectedIconFile.name}（
                    {Math.round(
                      selectedIconFile.size /
                        1024,
                    )}
                    KB）
                  </span>

                  {handlers.onClearLocalIconFile && (
                    <button
                      type="button"
                      className="token-blueprint-card__icon-clear-btn"
                      onClick={() => {
                        handlers.onClearLocalIconFile?.();
                      }}
                      aria-label="選択したアイコンを取り消す"
                    >
                      <X size={16} />
                    </button>
                  )}
                </div>
              )}
          </div>

          <div className="token-blueprint-card__spacer">
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">
                トークン名
              </Label>

              {vm.isEditMode ? (
                <Input
                  value={vm.name}
                  placeholder="例：LUMINA VIP 会員トークン"
                  onChange={(event) => {
                    if (
                      !isIdentityLocked
                    ) {
                      handlers.onChangeName?.(
                        event.target.value,
                      );
                    }
                  }}
                  readOnly={
                    isIdentityLocked
                  }
                  className={`token-blueprint-card__readonly-input ${
                    isIdentityLocked
                      ? "readonly"
                      : ""
                  }`}
                />
              ) : (
                <div className="token-blueprint-card__view-value">
                  {vm.name ||
                    "未設定"}
                </div>
              )}
            </div>

            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">
                シンボル
              </Label>

              {vm.isEditMode ? (
                <Input
                  value={vm.symbol}
                  placeholder="例：LUMI"
                  onChange={(event) => {
                    if (
                      !isIdentityLocked
                    ) {
                      handlers.onChangeSymbol?.(
                        event.target.value.toUpperCase(),
                      );
                    }
                  }}
                  readOnly={
                    isIdentityLocked
                  }
                  className={`token-blueprint-card__readonly-input ${
                    isIdentityLocked
                      ? "readonly"
                      : ""
                  }`}
                />
              ) : (
                <div className="token-blueprint-card__view-value">
                  {vm.symbol ||
                    "未設定"}
                </div>
              )}
            </div>

            <div className="token-blueprint-card__brand-label-cell">
              <Label className="token-blueprint-card__label">
                ブランド
              </Label>

              {vm.isEditMode &&
              !isIdentityLocked ? (
                <Popover>
                  <PopoverTrigger>
                    <div
                      className="token-blueprint-card__select"
                      role="button"
                      aria-label="ブランドを選択"
                    >
                      <Input
                        readOnly
                        value={
                          vm.brandName ||
                          vm.brandId ||
                          "ブランド未設定"
                        }
                        className="token-blueprint-card__select-input"
                      />
                    </div>
                  </PopoverTrigger>

                  <PopoverContent
                    align="start"
                    className="token-blueprint-card__popover"
                  >
                    {vm.brandOptions.length ===
                    0 ? (
                      <div className="token-blueprint-card__popover-empty">
                        ブランド候補が未設定です
                      </div>
                    ) : (
                      <div className="token-blueprint-card__popover-list">
                        {vm.brandOptions.map(
                          (brand) => (
                            <button
                              key={brand.id}
                              type="button"
                              className={
                                "token-blueprint-card__popover-item" +
                                (brand.id ===
                                vm.brandId
                                  ? " is-active"
                                  : "")
                              }
                              onClick={() => {
                                handlers.onChangeBrand?.(
                                  brand.id,
                                  brand.name,
                                );
                              }}
                            >
                              {brand.name}
                            </button>
                          ),
                        )}
                      </div>
                    )}
                  </PopoverContent>
                </Popover>
              ) : (
                <Input
                  readOnly
                  value={
                    vm.brandName ||
                    vm.brandId ||
                    "ブランド未設定"
                  }
                  className="token-blueprint-card__readonly-input readonly"
                />
              )}
            </div>

            {isIdentityLocked && (
              <div className="token-blueprint-card__identity-lock-message">
                このトークン設計はmint済みのため、
                トークン名・シンボル・ブランドは変更できません。
              </div>
            )}
          </div>
        </div>

        <div className="token-blueprint-card__description">
          <Label className="token-blueprint-card__label">
            説明
          </Label>

          {vm.isEditMode ? (
            <textarea
              ref={
                handlers.descriptionRef ??
                undefined
              }
              value={vm.description}
              placeholder="このトークンで付与する権利・特典を記載してください。"
              onChange={(event) => {
                handlers.onChangeDescription?.(
                  event.target.value,
                );
              }}
              className="token-blueprint-card__description-input"
            />
          ) : (
            <div className="token-blueprint-card__description-value">
              {vm.description ||
                "未設定"}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
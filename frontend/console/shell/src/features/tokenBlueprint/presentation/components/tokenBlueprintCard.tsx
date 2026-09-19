// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenBlueprintCard.tsx

import * as React from "react";
import { Link2, Upload, X } from "lucide-react";

import { IMAGE_STORAGE_ACCEPT } from "../../../../shared/storage/imageStoragePolicy";
import type { IconCropPosition } from "../../../../shared/types/iconCrop";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardBadge,
  CardButton,
  CardContent,
  CardField,
  CardFields,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardInput,
  CardLabel,
  CardReadonly,
  CardSelectWrap,
  CardTextarea,
  CardTitle,
  CardViewValue,
} from "../../../../shared/ui/card";
import IconCropper from "../../../../shared/ui/icon-cropper";
import EntityIcon from "../../../../shared/ui/icon";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shared/ui/popover";
import Text from "../../../../shared/ui/text";

export type TokenBlueprintCardViewModel = {
  id: string;
  name: string;
  symbol: string;
  brandId: string;
  brandName: string;
  description: string;
  iconUrl?: string;

  // mint済みの場合に、トークン名・シンボル・ブランドを編集不可にするための判定値。
  minted: boolean;

  // UIで選択されたアイコンファイル。
  iconFile?: File | null;

  // アイコン画像の切り抜き状態。
  iconCropPosition: IconCropPosition;
  iconCropScale: number;

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
  onChangeBrand?: (id: string, name: string) => void;
  onChangeDescription?: (value: string) => void;

  descriptionRef?: React.RefObject<HTMLTextAreaElement>;
  iconInputRef?: React.RefObject<HTMLInputElement>;
  onRequestPickIconFile?: () => void;
  onIconInputChange?: (event: React.ChangeEvent<HTMLInputElement>) => void;

  onIconCropPositionChange?: (position: IconCropPosition) => void;
  onIconCropScaleChange?: (scale: number) => void;
  onIconCropViewportSizeChange?: (size: number) => void;

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
  const canEditIcon = vm.isEditMode;

  /**
   * mint済みトークンでは、editモードへ移行しても
   * トークン名・シンボル・ブランドを変更不可にする。
   */
  const isIdentityLocked = Boolean(vm.isEditMode && vm.minted);
  const selectedIconFile = vm.iconFile ?? null;
  const isCroppingIcon = Boolean(canEditIcon && selectedIconFile && vm.iconUrl);

  return (
    <Card elevated largeRadius>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon variant="primary">
            <Link2 className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong truncate>
            {vm.id ? "トークン設計" : "トークン：新規トークン設計"}
          </CardTitle>

          <CardBadge variant="primary">
            設計情報
          </CardBadge>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent size="large">
        <div className="token-blueprint-card__top">
          <div className="token-blueprint-card__icon-area">
            {isCroppingIcon ? (
              <IconCropper
                src={vm.iconUrl ?? ""}
                position={vm.iconCropPosition}
                scale={vm.iconCropScale}
                onPositionChange={(position) => {
                  handlers.onIconCropPositionChange?.(position);
                }}
                onScaleChange={(scale) => {
                  handlers.onIconCropScaleChange?.(scale);
                }}
                onViewportSizeChange={(size) => {
                  handlers.onIconCropViewportSizeChange?.(size);
                }}
                alt="トークンアイコンの切り抜きプレビュー"
              />
            ) : (
              <EntityIcon
                src={vm.iconUrl}
                name={vm.name}
                alt="トークンアイコン"
                size="fluid"
                variant="upload"
                fallback={
                  canEditIcon ? (
                    <>
                      アイコン画像を
                      <br />
                      アップロード
                    </>
                  ) : (
                    "アイコン未設定"
                  )
                }
                onClick={
                  canEditIcon
                    ? () => {
                        handlers.onRequestPickIconFile?.();
                      }
                    : undefined
                }
              />
            )}

            {canEditIcon ? (
              <>
                <input
                  ref={handlers.iconInputRef ?? undefined}
                  type="file"
                  accept={IMAGE_STORAGE_ACCEPT}
                  hidden
                  onChange={handlers.onIconInputChange}
                />

                <CardButton
                  variant="primary"
                  className="token-blueprint-card__upload-btn"
                  onClick={() => {
                    handlers.onRequestPickIconFile?.();
                  }}
                >
                  <Upload className="card__button-icon" />
                  アップロード
                </CardButton>
              </>
            ) : null}

            {canEditIcon && selectedIconFile ? (
              <div className="token-blueprint-card__icon-selected">
                <span>
                  選択中：{selectedIconFile.name}（
                  {Math.round(selectedIconFile.size / 1024)}
                  KB）
                </span>

                {handlers.onClearLocalIconFile ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      handlers.onClearLocalIconFile?.();
                    }}
                    aria-label="選択したアイコンを取り消す"
                    title="選択を取り消す"
                  >
                    <X aria-hidden="true" />
                  </Button>
                ) : null}
              </div>
            ) : null}
          </div>

          <CardFields>
            <CardField>
              <CardLabel strong>
                トークン名
              </CardLabel>

              {vm.isEditMode ? (
                isIdentityLocked ? (
                  <CardReadonly inputLike size="large">
                    {vm.name || "未設定"}
                  </CardReadonly>
                ) : (
                  <CardInput
                    sizeVariant="large"
                    value={vm.name}
                    placeholder="例：LUMINA VIP 会員トークン"
                    onChange={(event) => {
                      handlers.onChangeName?.(event.target.value);
                    }}
                  />
                )
              ) : (
                <CardViewValue>
                  {vm.name || "未設定"}
                </CardViewValue>
              )}
            </CardField>

            <CardField>
              <CardLabel strong>
                シンボル
              </CardLabel>

              {vm.isEditMode ? (
                isIdentityLocked ? (
                  <CardReadonly inputLike size="large">
                    {vm.symbol || "未設定"}
                  </CardReadonly>
                ) : (
                  <CardInput
                    sizeVariant="large"
                    value={vm.symbol}
                    placeholder="例：LUMI"
                    onChange={(event) => {
                      handlers.onChangeSymbol?.(
                        event.target.value.toUpperCase(),
                      );
                    }}
                  />
                )
              ) : (
                <CardViewValue>
                  {vm.symbol || "未設定"}
                </CardViewValue>
              )}
            </CardField>

            <CardField full>
              <CardLabel strong>
                ブランド
              </CardLabel>

              {vm.isEditMode && !isIdentityLocked ? (
                <Popover>
                  <PopoverTrigger>
                    <CardSelectWrap
                      role="button"
                      aria-label="ブランドを選択"
                    >
                      <CardInput
                        sizeVariant="large"
                        readOnly
                        value={vm.brandName || vm.brandId || "ブランド未設定"}
                      />
                    </CardSelectWrap>
                  </PopoverTrigger>

                  <PopoverContent
                    align="start"
                    className="popover__content--compact popover__content--medium"
                  >
                    {vm.brandOptions.length === 0 ? (
                      <div className="popover__empty">
                        ブランド候補が未設定です
                      </div>
                    ) : (
                      <div className="popover__list">
                        {vm.brandOptions.map((brand) => (
                          <button
                            key={brand.id}
                            type="button"
                            className={
                              "popover__item" +
                              (brand.id === vm.brandId ? " is-active" : "")
                            }
                            onClick={() => {
                              handlers.onChangeBrand?.(brand.id, brand.name);
                            }}
                          >
                            {brand.name}
                          </button>
                        ))}
                      </div>
                    )}
                  </PopoverContent>
                </Popover>
              ) : (
                <CardReadonly inputLike size="large">
                  {vm.brandName || vm.brandId || "ブランド未設定"}
                </CardReadonly>
              )}
            </CardField>

            {isIdentityLocked ? (
              <Text
                as="div"
                size="xs"
                tone="muted"
                className="token-blueprint-card__identity-lock-message"
              >
                このトークン設計はmint済みのため、トークン名・シンボル・ブランドは変更できません。
              </Text>
            ) : null}
          </CardFields>
        </div>

        <CardField className="token-blueprint-card__description">
          <CardLabel strong>
            説明
          </CardLabel>

          {vm.isEditMode ? (
            <CardTextarea
              ref={handlers.descriptionRef ?? undefined}
              value={vm.description}
              placeholder="このトークンで付与する権利・特典を記載してください。"
              onChange={(event) => {
                handlers.onChangeDescription?.(event.target.value);
              }}
            />
          ) : (
            <CardViewValue className="token-blueprint-card__description-value">
              {vm.description || "未設定"}
            </CardViewValue>
          )}
        </CardField>
      </CardContent>
    </Card>
  );
}
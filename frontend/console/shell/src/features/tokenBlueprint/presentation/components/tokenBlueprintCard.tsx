// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenBlueprintCard.tsx

import * as React from "react";
import { Link2 } from "lucide-react";

import { IMAGE_STORAGE_ACCEPT } from "../../../../shared/storage/imageStoragePolicy";
import type { IconCropPosition } from "../../../../shared/types/iconCrop";
import {
  Card,
  CardBadge,
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
import MediaUploader from "../../../../shared/ui/mediaUploader";
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
  minted: boolean;
  iconFile?: File | null;
  iconCropPosition: IconCropPosition;
  iconCropScale: number;
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
  onIconFilesSelected?: (files: File[]) => void;
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
  const canEditIcon = vm.isEditMode;
  const isIdentityLocked = Boolean(vm.isEditMode && vm.minted);
  const selectedIconFile = vm.iconFile ?? null;
  const isCroppingIcon = Boolean(
    canEditIcon && selectedIconFile && vm.iconUrl,
  );

  const iconItems = vm.iconUrl
    ? [
        {
          id: "token-blueprint-icon",
          src: vm.iconUrl,
          name: selectedIconFile
            ? `選択中：${selectedIconFile.name}（${Math.round(
                selectedIconFile.size / 1024,
              )}KB）`
            : undefined,
          alt: "トークンアイコン",
          type: "image" as const,
          contentType: selectedIconFile?.type,
        },
      ]
    : [];

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
          <MediaUploader
            items={iconItems}
            accept={IMAGE_STORAGE_ACCEPT}
            variant="single"
            pickerVariant="button"
            title={null}
            showCount={false}
            showFileNames={Boolean(selectedIconFile)}
            showPicker={canEditIcon}
            showRemoveButton={Boolean(selectedIconFile)}
            previewFramed={false}
            pickerLabel="アップロード"
            replaceLabel="アイコンを変更"
            className="token-blueprint-card__icon-area"
            disabled={!canEditIcon}
            renderPreview={(_item, { openPicker }) =>
              isCroppingIcon ? (
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
                  fallback="アイコン未設定"
                  onClick={canEditIcon ? openPicker : undefined}
                />
              )
            }
            renderEmpty={({ openPicker }) => (
              <EntityIcon
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
                onClick={canEditIcon ? openPicker : undefined}
              />
            )}
            onFilesSelected={(files) => {
              handlers.onIconFilesSelected?.(files);
            }}
            onRemove={() => {
              handlers.onClearLocalIconFile?.();
            }}
          />

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
                        value={
                          vm.brandName ||
                          vm.brandId ||
                          "ブランド未設定"
                        }
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
                              (brand.id === vm.brandId
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
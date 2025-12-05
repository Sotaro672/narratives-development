// frontend/console/tokenBlueprint/src/presentation/components/tokenBlueprintCard.tsx
import * as React from "react";
import { Link2, Upload, Eye } from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Input } from "../../../../shell/src/shared/ui/input";
import { Badge } from "../../../../shell/src/shared/ui/badge";
import { Label } from "../../../../shell/src/shared/ui/label";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

import "../styles/tokenBlueprint.css";

export type TokenBlueprintCardViewModel = {
  id: string;
  name: string;
  symbol: string;
  brandId: string;
  brandName: string;
  description: string;
  iconUrl?: string;
  isEditMode: boolean;
  brandOptions: { id: string; name: string }[];
};

export type TokenBlueprintCardHandlers = {
  onChangeName?: (v: string) => void;
  onChangeSymbol?: (v: string) => void;
  onChangeBrand?: (id: string, name: string) => void;
  onChangeDescription?: (v: string) => void;

  onUploadIcon?: () => void;
  onPreview?: () => void;
  onToggleEditMode?: () => void;

  // 外部からモード制御
  setEditMode?: (edit: boolean) => void;
  reset?: () => void;
};

export default function TokenBlueprintCard({
  vm,
  handlers,
}: {
  vm: TokenBlueprintCardViewModel;
  handlers: TokenBlueprintCardHandlers;
}) {
  const descriptionRef = React.useRef<HTMLTextAreaElement | null>(null);

  React.useEffect(() => {
    if (!descriptionRef.current) return;
    descriptionRef.current.style.height = "auto";
    descriptionRef.current.style.height = `${descriptionRef.current.scrollHeight}px`;
  }, [vm.description]);

  return (
    <Card className="token-blueprint-card">
      <CardHeader className="token-blueprint-card__header">
        <div className="token-blueprint-card__header-left">
          <span className="token-blueprint-card__header-icon">
            <Link2 className="token-blueprint-card__link-icon" />
          </span>
          <CardTitle className="token-blueprint-card__header-title">
            {vm.id ? "トークン設計" : "トークン：新規トークン設計"}
          </CardTitle>
          <Badge className="token-blueprint-card__header-badge">設計情報</Badge>
        </div>

        <button
          type="button"
          className="token-blueprint-card__preview-btn"
          onClick={() => handlers.onPreview?.()}
          aria-label="プレビュー"
        >
          <Eye className="token-blueprint-card__preview-icon" />
        </button>
      </CardHeader>

      <CardContent>
        <div className="token-blueprint-card__top">
          {/* アイコン */}
          <div className="token-blueprint-card__icon-area">
            <div className="token-blueprint-card__icon-wrap">
              {vm.iconUrl ? (
                <img src={vm.iconUrl} alt="Token Icon" />
              ) : (
                <div className="token-blueprint-card__icon-placeholder">
                  アイコン画像を
                  <br />
                  アップロード
                </div>
              )}
            </div>

            {/* 編集モードのみ */}
            {vm.isEditMode && (
              <button
                type="button"
                className="token-blueprint-card__upload-btn"
                onClick={() => handlers.onUploadIcon?.()}
              >
                <Upload className="token-blueprint-card__upload-icon" />
                アップロード
              </button>
            )}
          </div>

          {/* 右側フィールド */}
          <div className="token-blueprint-card__spacer">
            {/* トークン名 */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">トークン名</Label>
              <Input
                value={vm.name}
                placeholder="例：LUMINA VIP 会員トークン"
                onChange={(e) =>
                  vm.isEditMode && handlers.onChangeName?.(e.target.value)
                }
                readOnly={!vm.isEditMode}
                className={`token-blueprint-card__readonly-input ${
                  !vm.isEditMode ? "readonly" : ""
                }`}
              />
            </div>

            {/* シンボル */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">シンボル</Label>
              <Input
                value={vm.symbol}
                placeholder="例：LUMI"
                onChange={(e) =>
                  vm.isEditMode &&
                  handlers.onChangeSymbol?.(e.target.value.toUpperCase())
                }
                readOnly={!vm.isEditMode}
                className={`token-blueprint-card__readonly-input ${
                  !vm.isEditMode ? "readonly" : ""
                }`}
              />
            </div>

            {/* ブランド */}
            <div className="token-blueprint-card__brand-label-cell">
              <Label className="token-blueprint-card__label">ブランド</Label>

              {vm.isEditMode ? (
                <Popover>
                  <PopoverTrigger>
                    <div className="token-blueprint-card__select">
                      <Input
                        readOnly
                        value={vm.brandName || vm.brandId || "ブランド未設定"}
                        className="token-blueprint-card__select-input"
                      />
                      <span className="token-blueprint-card__select-caret">▾</span>
                    </div>
                  </PopoverTrigger>

                  <PopoverContent
                    align="start"
                    className="token-blueprint-card__popover"
                  >
                    {vm.brandOptions.length === 0 && (
                      <div className="token-blueprint-card__popover-empty">
                        ブランド候補が未設定です
                      </div>
                    )}

                    {vm.brandOptions.map((b) => (
                      <button
                        key={b.id}
                        type="button"
                        className={
                          "token-blueprint-card__popover-item" +
                          (b.id === vm.brandId ? " is-active" : "")
                        }
                        onClick={() => handlers.onChangeBrand?.(b.id, b.name)}
                      >
                        {b.name}
                      </button>
                    ))}
                  </PopoverContent>
                </Popover>
              ) : (
                <Input
                  readOnly
                  value={vm.brandName || vm.brandId || "ブランド未設定"}
                  className="token-blueprint-card__readonly-input readonly"
                />
              )}
            </div>
          </div>
        </div>

        {/* 説明 */}
        <div className="token-blueprint-card__description">
          <Label className="token-blueprint-card__label">説明</Label>

          <textarea
            ref={descriptionRef}
            value={vm.description}
            placeholder="このトークンで付与する権利・特典を記載してください。"
            onChange={(e) =>
              vm.isEditMode && handlers.onChangeDescription?.(e.target.value)
            }
            readOnly={!vm.isEditMode}
            className={`token-blueprint-card__description-input ${
              !vm.isEditMode ? "readonly" : ""
            }`}
          />
        </div>
      </CardContent>
    </Card>
  );
}

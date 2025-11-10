// frontend/tokenBlueprint/src/presentation/components/tokenBlueprintCard.tsx

import * as React from "react";
import { Link2, Upload, Calendar, Eye } from "lucide-react";

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

import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";
import "../styles/tokenBlueprint.css";

type TokenBlueprintCardProps = {
  /** 初期表示モード（true: 編集、false: 閲覧） */
  initialEditMode?: boolean;

  /**
   * TokenBlueprint エンティティに対応した初期値。
   * brandName は brandId をブランドマスタから解決した表示用ラベル想定。
   */
  initialTokenBlueprint?: Partial<TokenBlueprint> & {
    brandName?: string;
  };

  /** ドメイン外の追加表示項目（例: 焼却予定日） */
  initialBurnAt?: string;

  /** 表示用アイコンURL（iconId から解決済みを想定。任意） */
  initialIconUrl?: string;
};

export default function TokenBlueprintCard({
  initialEditMode = false,
  initialTokenBlueprint,
  initialBurnAt,
  initialIconUrl,
}: TokenBlueprintCardProps) {
  const tb = initialTokenBlueprint ?? {};

  // TokenBlueprint スキーマに対応した状態
  const [id] = React.useState(tb.id ?? "");
  const [name, setName] = React.useState(tb.name ?? "");
  const [symbol, setSymbol] = React.useState(tb.symbol ?? "");
  const [brandId, setBrandId] = React.useState(tb.brandId ?? "");
  const [brandName, setBrandName] = React.useState(tb.brandName ?? "");
  const [description, setDescription] = React.useState(tb.description ?? "");
  const [burnAt, setBurnAt] = React.useState(initialBurnAt ?? "");
  const [iconUrl] = React.useState(initialIconUrl ?? "");
  const [isEditMode] = React.useState(initialEditMode);

  // brandId/brandName は本来 brandService 等で解決する想定。
  // ここではプレースホルダとして空配列。
  const brandOptions: { id: string; name: string }[] = [];

  // 説明欄の自動リサイズ
  const descriptionRef = React.useRef<HTMLTextAreaElement | null>(null);

  const autoResizeDescription = React.useCallback(() => {
    const el = descriptionRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, []);

  React.useEffect(() => {
    autoResizeDescription();
  }, [description, autoResizeDescription]);

  const handleUploadClick = () => {
    if (!isEditMode) return;
    alert("トークンアイコンのアップロード（モック）");
  };

  const handlePreview = () => {
    alert("プレビュー画面を開きます（モック）");
  };

  const displayBrand = brandName || brandId || "ブランド未設定";

  return (
    <Card className="token-blueprint-card">
      <CardHeader className="token-blueprint-card__header">
        <div className="token-blueprint-card__header-left">
          <span className="token-blueprint-card__header-icon">
            <Link2 className="token-blueprint-card__link-icon" />
          </span>
          <CardTitle className="token-blueprint-card__header-title">
            {id ? `トークン：${id}` : "トークン：新規トークン設計"}
          </CardTitle>
          <Badge className="token-blueprint-card__header-badge">
            設計情報
          </Badge>
        </div>

        <button
          type="button"
          className="token-blueprint-card__preview-btn"
          onClick={handlePreview}
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
              {iconUrl ? (
                <img src={iconUrl} alt="Token Icon" />
              ) : (
                <div className="token-blueprint-card__icon-placeholder">
                  アイコン画像を
                  <br />
                  アップロード
                </div>
              )}
            </div>

            {/* 編集モードのみアップロードボタン表示 */}
            {isEditMode && (
              <button
                type="button"
                className="token-blueprint-card__upload-btn"
                onClick={handleUploadClick}
              >
                <Upload className="token-blueprint-card__upload-icon" />
                アップロード
              </button>
            )}
          </div>

          {/* 右側フィールド */}
          <div className="token-blueprint-card__spacer">
            {/* トークン名 (TokenBlueprint.name) */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">トークン名</Label>
              <div
                className={`token-blueprint-card__readonly ${
                  !isEditMode ? "readonly-mode" : ""
                }`}
              >
                <Input
                  value={name}
                  placeholder="例：LUMINA VIP 会員トークン"
                  onChange={(e) =>
                    isEditMode && setName(e.target.value)
                  }
                  readOnly={!isEditMode}
                  className={`token-blueprint-card__readonly-input ${
                    !isEditMode ? "readonly" : ""
                  }`}
                />
              </div>
            </div>

            {/* シンボル (TokenBlueprint.symbol) */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">シンボル</Label>
              <div
                className={`token-blueprint-card__readonly ${
                  !isEditMode ? "readonly-mode" : ""
                }`}
              >
                <Input
                  value={symbol}
                  placeholder="例：LUMI"
                  onChange={(e) =>
                    isEditMode && setSymbol(e.target.value.toUpperCase())
                  }
                  readOnly={!isEditMode}
                  className={`token-blueprint-card__readonly-input ${
                    !isEditMode ? "readonly" : ""
                  }`}
                />
              </div>
            </div>

            {/* ブランド (TokenBlueprint.brandId → 表示名) */}
            <div className="token-blueprint-card__brand-label-cell">
              <Label className="token-blueprint-card__label">ブランド</Label>
              {isEditMode ? (
                <Popover>
                  <PopoverTrigger>
                    <div className="token-blueprint-card__select">
                      <Input
                        readOnly
                        value={displayBrand}
                        placeholder="ブランドを選択"
                        className="token-blueprint-card__select-input"
                      />
                      <span className="token-blueprint-card__select-caret">
                        ▾
                      </span>
                    </div>
                  </PopoverTrigger>
                  <PopoverContent
                    align="start"
                    className="token-blueprint-card__popover"
                  >
                    {brandOptions.length === 0 && (
                      <div className="token-blueprint-card__popover-empty">
                        ブランド候補が未設定です
                      </div>
                    )}
                    {brandOptions.map((b) => (
                      <button
                        key={b.id}
                        type="button"
                        className={
                          "token-blueprint-card__popover-item" +
                          (b.id === brandId ? " is-active" : "")
                        }
                        onClick={() => {
                          setBrandId(b.id);
                          setBrandName(b.name);
                        }}
                      >
                        {b.name}
                      </button>
                    ))}
                  </PopoverContent>
                </Popover>
              ) : (
                <Input
                  readOnly
                  value={displayBrand}
                  placeholder="ブランド未設定"
                  className="token-blueprint-card__readonly-input readonly"
                />
              )}
            </div>
          </div>
        </div>

        {/* 説明 (TokenBlueprint.description) */}
        <div className="token-blueprint-card__description">
          <Label className="token-blueprint-card__label">説明</Label>
          <div
            className={`token-blueprint-card__description-box ${
              !isEditMode ? "readonly-mode" : ""
            }`}
          >
            <textarea
              ref={descriptionRef}
              value={description}
              placeholder="このトークンで付与する権利・特典、利用条件などを記載してください。"
              onChange={(e) => {
                if (!isEditMode) return;
                setDescription(e.target.value);
              }}
              readOnly={!isEditMode}
              className={`token-blueprint-card__description-input ${
                !isEditMode ? "readonly" : ""
              }`}
            />
          </div>
        </div>

        {/* 焼却予定日（ドメイン外拡張。任意） */}
        <div className="token-blueprint-card__expires">
          <Label className="token-blueprint-card__label">焼却予定日</Label>
          <div className="token-blueprint-card__expires-row">
            <Input
              type="date"
              value={burnAt}
              onChange={(e) =>
                isEditMode && setBurnAt(e.target.value)
              }
              readOnly={!isEditMode}
              className={`token-blueprint-card__expires-input ${
                !isEditMode ? "readonly" : ""
              }`}
            />
            <div className="token-blueprint-card__calendar-icon">
              <Calendar className="token-blueprint-card__calendar-icon-svg" />
            </div>
          </div>
          {!burnAt && (
            <div className="token-blueprint-card__expires-hint">
              任意。キャンペーン終了日など、トークンの有効期限がある場合のみ設定します。
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

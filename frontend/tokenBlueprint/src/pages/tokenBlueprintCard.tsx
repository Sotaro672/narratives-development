// frontend/tokenBlueprint/src/pages/tokenBlueprintCard.tsx
import * as React from "react";
import { Link2, Upload, Calendar, Eye } from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui/card";
import { Input } from "../../../shared/ui/input";
import { Badge } from "../../../shared/ui/badge";
import { Label } from "../../../shared/ui/label";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../shared/ui/popover";

import "./tokenBlueprintCard.css";

type TokenBlueprintCardProps = {
  /** 初期表示を編集モードにするか（既定: false） */
  initialEditMode?: boolean;

  /** 各項目の初期値（未指定の場合は空で表示） */
  initialTokenBlueprintId?: string;
  initialTokenName?: string;
  initialSymbol?: string;
  initialBrand?: string;
  initialDescription?: string;
  initialBurnAt?: string;
  initialIconUrl?: string;
};

export default function TokenBlueprintCard({
  initialEditMode = false,
  initialTokenBlueprintId,
  initialTokenName,
  initialSymbol,
  initialBrand,
  initialDescription,
  initialBurnAt,
  initialIconUrl,
}: TokenBlueprintCardProps) {
  const [tokenBlueprintId] = React.useState(initialTokenBlueprintId ?? "");
  const [tokenName, setTokenName] = React.useState(initialTokenName ?? "");
  const [symbol, setSymbol] = React.useState(initialSymbol ?? "");
  const [brand, setBrand] = React.useState(initialBrand ?? "");
  const [description, setDescription] = React.useState(initialDescription ?? "");
  const [burnAt, setBurnAt] = React.useState(initialBurnAt ?? "");
  const [iconUrl, setIconUrl] = React.useState(initialIconUrl ?? "");
  const [isEditMode] = React.useState(initialEditMode);

  const brandOptions: string[] = [];

  const handleUploadClick = () => {
    if (!isEditMode) return;
    alert("トークンアイコンのアップロード（モック）");
  };

  const handlePreview = () => {
    alert("プレビュー画面を開きます（モック）");
  };

  return (
    <Card className="token-blueprint-card">
      {/* ヘッダー：左にタイトル、右にプレビューボタン（横並び） */}
      <CardHeader className="token-blueprint-card__header">
        <div className="token-blueprint-card__header-left">
          <span className="token-blueprint-card__header-icon">
            <Link2 className="token-blueprint-card__link-icon" />
          </span>
          <CardTitle className="token-blueprint-card__header-title">
            {tokenBlueprintId
              ? `トークン：${tokenBlueprintId}`
              : "トークン：新規トークン設計"}
          </CardTitle>
          <Badge className="token-blueprint-card__header-badge">設計情報</Badge>
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
        {/* 上部：左にアイコン、右に縦並びフィールド */}
        <div className="token-blueprint-card__top">
          {/* 左：アイコン＋アップロード */}
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
            <button
              type="button"
              className={`token-blueprint-card__upload-btn ${
                !isEditMode ? "disabled" : ""
              }`}
              onClick={handleUploadClick}
              disabled={!isEditMode}
            >
              <Upload className="token-blueprint-card__upload-icon" />
              アップロード
            </button>
          </div>

          {/* 右：トークン名 / シンボル / ブランド */}
          <div className="token-blueprint-card__spacer">
            {/* トークン名 */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">トークン名</Label>
              <div className="token-blueprint-card__readonly">
                <Input
                  value={tokenName}
                  placeholder="例：LUMINA VIP 会員トークン"
                  onChange={(e) =>
                    isEditMode && setTokenName(e.target.value)
                  }
                  readOnly={!isEditMode}
                  className={`token-blueprint-card__readonly-input ${
                    !isEditMode ? "readonly" : ""
                  }`}
                />
              </div>
            </div>

            {/* シンボル */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">シンボル</Label>
              <div className="token-blueprint-card__readonly">
                <Input
                  value={symbol}
                  placeholder="例：LUMI"
                  onChange={(e) => isEditMode && setSymbol(e.target.value)}
                  readOnly={!isEditMode}
                  className={`token-blueprint-card__readonly-input ${
                    !isEditMode ? "readonly" : ""
                  }`}
                />
              </div>
            </div>

            {/* ブランド：Input + Popover */}
            <div className="token-blueprint-card__brand-label-cell">
              <Label className="token-blueprint-card__label">ブランド</Label>
              {isEditMode ? (
                <Popover>
                  <PopoverTrigger>
                    <div className="token-blueprint-card__select">
                      <Input
                        readOnly
                        value={brand}
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
                        key={b}
                        type="button"
                        className={
                          "token-blueprint-card__popover-item" +
                          (b === brand ? " is-active" : "")
                        }
                        onClick={() => setBrand(b)}
                      >
                        {b}
                      </button>
                    ))}
                  </PopoverContent>
                </Popover>
              ) : (
                <Input
                  readOnly
                  value={brand}
                  placeholder="ブランド未設定"
                  className="token-blueprint-card__readonly-input readonly"
                />
              )}
            </div>
          </div>
        </div>

        {/* 説明 */}
        <div className="token-blueprint-card__description">
          <Label className="token-blueprint-card__label">説明</Label>
          <div className="token-blueprint-card__description-box">
            <textarea
              value={description}
              placeholder="このトークンで付与する権利・特典、利用条件などを記載してください。"
              onChange={(e) =>
                isEditMode && setDescription(e.target.value)
              }
              readOnly={!isEditMode}
              className={`token-blueprint-card__description-input ${
                !isEditMode ? "readonly" : ""
              }`}
            />
          </div>
        </div>

        {/* 焼却予定日 */}
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

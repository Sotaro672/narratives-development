import * as React from "react";
import { Link2, Upload, Calendar, Pencil, Save } from "lucide-react";

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

import { TOKEN_BLUEPRINTS } from "../../mockdata";
import "./tokenBlueprintCard.css";

// ✅ initialEditMode を受け取れるようにする
type TokenBlueprintCardProps = {
  initialEditMode?: boolean;
};

export default function TokenBlueprintCard({
  initialEditMode = false,
}: TokenBlueprintCardProps) {
  // ─────────────────────────────────────────────
  // モックデータを利用（ここでは 0 件目を利用）
  // ─────────────────────────────────────────────
  const blueprint = TOKEN_BLUEPRINTS[0];

  // ─────────────────────────────────────────────
  // 状態管理
  // ─────────────────────────────────────────────
  const [tokenBlueprintId] = React.useState(blueprint.tokenBlueprintId);
  const [tokenName, setTokenName] = React.useState(blueprint.name);
  const [symbol, setSymbol] = React.useState(blueprint.symbol);
  const [brand, setBrand] = React.useState(blueprint.brand);
  const [description, setDescription] = React.useState(blueprint.description);
  const [burnAt, setBurnAt] = React.useState(blueprint.burnAt);
  const [iconUrl, setIconUrl] = React.useState(blueprint.iconUrl);
  const [isEditMode, setIsEditMode] = React.useState(initialEditMode);

  const brandOptions = Array.from(
    new Set(TOKEN_BLUEPRINTS.map((b) => b.brand))
  );

  const handleUploadClick = () => {
    if (!isEditMode) return;
    alert("トークンアイコンのアップロード（モック）");
  };

  const handleSave = () => {
    // 実際はここで API に送信する想定
    console.log("保存内容:", {
      tokenBlueprintId,
      tokenName,
      symbol,
      brand,
      description,
      burnAt,
      iconUrl,
    });
    alert("変更を保存しました（モック）");
    setIsEditMode(false);
  };

  return (
    <Card className="token-blueprint-card">
      {/* ヘッダー：トークンID */}
      <CardHeader className="token-blueprint-card__header">
        <span className="token-blueprint-card__header-icon">
          <Link2 className="token-blueprint-card__link-icon" />
        </span>
        <CardTitle className="token-blueprint-card__header-title">
          トークン：{tokenBlueprintId}
        </CardTitle>
        <Badge className="token-blueprint-card__header-badge">設計情報</Badge>

        {/* 編集モード切替ボタン */}
        <button
          type="button"
          onClick={() => (isEditMode ? handleSave() : setIsEditMode(true))}
          className="token-blueprint-card__edit-toggle-btn"
        >
          {isEditMode ? (
            <>
              <Save className="w-4 h-4 mr-1" /> 保存
            </>
          ) : (
            <>
              <Pencil className="w-4 h-4 mr-1" /> 編集
            </>
          )}
        </button>
      </CardHeader>

      <CardContent>
        {/* 上部：左にアイコン、右に縦並びフィールド */}
        <div className="token-blueprint-card__top">
          {/* 左：アイコン＋アップロード */}
          <div className="token-blueprint-card__icon-area">
            <div className="token-blueprint-card__icon-wrap">
              <img src={iconUrl} alt="Token Icon" />
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
                  onChange={(e) => setTokenName(e.target.value)}
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
                  onChange={(e) => setSymbol(e.target.value)}
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
                  className="token-blueprint-card__readonly-input readonly"
                />
              )}
            </div>
          </div>
        </div>

        {/* 説明：カード幅いっぱい */}
        <div className="token-blueprint-card__description">
          <Label className="token-blueprint-card__label">説明</Label>
          <div className="token-blueprint-card__description-box">
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
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
              onChange={(e) => setBurnAt(e.target.value)}
              readOnly={!isEditMode}
              className={`token-blueprint-card__expires-input ${
                !isEditMode ? "readonly" : ""
              }`}
            />
            <div className="token-blueprint-card__calendar-icon">
              <Calendar className="token-blueprint-card__calendar-icon-svg" />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// frontend/tokenBlueprint/src/pages/tokenBlueprintCard.tsx
import * as React from "react";
import { Link2, Upload, Calendar } from "lucide-react";

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

export default function TokenBlueprintCard() {
  const [tokenId] = React.useState("token_blueprint_001");
  const [tokenName, setTokenName] = React.useState("SILK Premium Token");
  const [symbol, setSymbol] = React.useState("SILK");
  const [brand, setBrand] = React.useState("LUMINA Fashion");
  const [description, setDescription] = React.useState(
    "プレミアムシルクブラウスの購入者限定トークン。限定コンテンツと特別割引をご利用いただけます。"
  );
  const [burnAt, setBurnAt] = React.useState("2025-12-31");
  const [iconUrl] = React.useState(
    "https://images.pexels.com/photos/8437005/pexels-photo-8437005.jpeg?auto=compress&cs=tinysrgb&w=300"
  );

  const brandOptions = ["LUMINA Fashion", "NEXUS Street"];

  const handleUploadClick = () => {
    alert("トークンアイコンのアップロード（モック）");
  };

  return (
    <Card className="token-blueprint-card">
      {/* ヘッダー：トークンID */}
      <CardHeader className="token-blueprint-card__header">
        <span className="token-blueprint-card__header-icon">
          <Link2 className="token-blueprint-card__link-icon" />
        </span>
        <CardTitle className="token-blueprint-card__header-title">
          トークン：{tokenId}
        </CardTitle>
        <Badge className="token-blueprint-card__header-badge">設計情報</Badge>
      </CardHeader>

      <CardContent>
        {/* 上部：左にアイコン、右(=spacer)に縦並びフィールド */}
        <div className="token-blueprint-card__top">
          {/* 左：アイコン＋アップロード */}
          <div className="token-blueprint-card__icon-area">
            <div className="token-blueprint-card__icon-wrap">
              <img src={iconUrl} alt="Token Icon" />
            </div>
            <button
              type="button"
              className="token-blueprint-card__upload-btn"
              onClick={handleUploadClick}
            >
              <Upload className="token-blueprint-card__upload-icon" />
              アップロード
            </button>
          </div>

          {/* 右：トークン名 / シンボル（縦並び） */}
          <div className="token-blueprint-card__spacer">
            {/* トークン名 */}
            <div className="token-blueprint-card__field-col">
              <Label className="token-blueprint-card__label">トークン名</Label>
              <div className="token-blueprint-card__readonly">
                <Input
                  value={tokenName}
                  onChange={(e) => setTokenName(e.target.value)}
                  className="token-blueprint-card__readonly-input"
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
                  className="token-blueprint-card__readonly-input"
                />
              </div>
            </div>
            {/* ブランド（アイコン右側の行：input + popover の両方） */}
            <div className="token-blueprint-card__brand-label-cell">
                <Label className="token-blueprint-card__label">ブランド</Label>
                <Popover>
                    <PopoverTrigger>
                    {/* Input風の見た目 + caret */}
                        <div className="token-blueprint-card__select">
                            <Input
                                readOnly
                                value={brand}
                                className="token-blueprint-card__select-input"
                            />
                            <span className="token-blueprint-card__select-caret">▾</span>
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
              className="token-blueprint-card__description-input"
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
              className="token-blueprint-card__expires-input"
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

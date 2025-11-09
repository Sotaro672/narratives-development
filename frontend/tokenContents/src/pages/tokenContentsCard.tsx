// frontend/tokenContents/src/pages/tokenContentsCard.tsx
import * as React from "react";
import {
  FileText,
  Upload,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui/card";
import { Button } from "../../../shared/ui/button";

import { MOCK_IMAGES } from "../../mockdata";
import "./tokenContentsCard.css";

type TokenContentsCardProps = {
  /** 初期表示する画像リスト。指定がなければモック、[]なら空表示。 */
  images?: string[];
};

export default function TokenContentsCard({ images }: TokenContentsCardProps) {
  const useExternal = images !== undefined;
  const list = useExternal ? images! : MOCK_IMAGES;

  const [index, setIndex] = React.useState(0);

  const hasImages = list.length > 0;

  const prev = () => {
    if (!hasImages) return;
    setIndex((i) => (i - 1 + list.length) % list.length);
  };

  const next = () => {
    if (!hasImages) return;
    setIndex((i) => (i + 1) % list.length);
  };

  const handleUpload = () => {
    alert("コンテンツファイルの追加（モック）");
  };

  return (
    <Card className="token-contents-card">
      <CardHeader className="token-contents-card__header">
        <div className="token-contents-card__title-wrap">
          <span className="token-contents-card__title-icon">
            <FileText className="token-contents-card__title-icon-svg" />
          </span>
          <CardTitle className="token-contents-card__title">
            コンテンツ
          </CardTitle>
        </div>

        <Button
          type="button"
          className="token-contents-card__add-btn"
          onClick={handleUpload}
        >
          <Upload className="token-contents-card__add-btn-icon" />
          ファイル追加
        </Button>
      </CardHeader>

      <CardContent>
        <div className="token-contents-card__viewer">
          {/* 左矢印 */}
          <button
            type="button"
            className="token-contents-card__nav token-contents-card__nav--left"
            onClick={prev}
            aria-label="前のコンテンツ"
            disabled={!hasImages}
          >
            <ChevronLeft className="token-contents-card__nav-icon" />
          </button>

          {/* 中央イメージスロット */}
          <div className="token-contents-card__image-slot">
            {hasImages ? (
              <img
                src={list[index]}
                alt={`コンテンツ ${index + 1}`}
                className="token-contents-card__image"
              />
            ) : (
              <div className="token-contents-card__placeholder">
                コンテンツがまだ登録されていません
              </div>
            )}
          </div>

          {/* 右矢印 */}
          <button
            type="button"
            className="token-contents-card__nav token-contents-card__nav--right"
            onClick={next}
            aria-label="次のコンテンツ"
            disabled={!hasImages}
          >
            <ChevronRight className="token-contents-card__nav-icon" />
          </button>
        </div>
      </CardContent>
    </Card>
  );
}

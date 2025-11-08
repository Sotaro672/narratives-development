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

import "./tokenContentsCard.css";

const MOCK_IMAGES = [
  "https://images.pexels.com/photos/373883/pexels-photo-373883.jpeg?auto=compress&cs=tinysrgb&w=800",
  "https://images.pexels.com/photos/1036856/pexels-photo-1036856.jpeg?auto=compress&cs=tinysrgb&w=800",
  "https://images.pexels.com/photos/3965545/pexels-photo-3965545.jpeg?auto=compress&cs=tinysrgb&w=800",
];

export default function TokenContentsCard() {
  const [index, setIndex] = React.useState(0);

  const prev = () => {
    setIndex((i) => (i - 1 + MOCK_IMAGES.length) % MOCK_IMAGES.length);
  };

  const next = () => {
    setIndex((i) => (i + 1) % MOCK_IMAGES.length);
  };

  const handleUpload = () => {
    alert("コンテンツファイルの追加（モック）");
  };

  return (
    <Card className="token-contents-card">
      {/* ヘッダー */}
      <CardHeader className="token-contents-card__header">
        <div className="token-contents-card__title-wrap">
          <span className="token-contents-card__title-icon">
            <FileText className="token-contents-card__title-icon-svg" />
          </span>
          <CardTitle className="token-contents-card__title">コンテンツ</CardTitle>
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

      {/* ビューア */}
      <CardContent>
        <div className="token-contents-card__viewer">
          {/* 左矢印 */}
          <button
            type="button"
            className="token-contents-card__nav token-contents-card__nav--left"
            onClick={prev}
            aria-label="前のコンテンツ"
          >
            <ChevronLeft className="token-contents-card__nav-icon" />
          </button>

          {/* 中央イメージスロット */}
          <div className="token-contents-card__image-slot">
            <img
              src={MOCK_IMAGES[index]}
              alt={`コンテンツ ${index + 1}`}
              className="token-contents-card__image"
            />
          </div>

          {/* 右矢印 */}
          <button
            type="button"
            className="token-contents-card__nav token-contents-card__nav--right"
            onClick={next}
            aria-label="次のコンテンツ"
          >
            <ChevronRight className="token-contents-card__nav-icon" />
          </button>
        </div>
      </CardContent>
    </Card>
  );
}

// frontend/tokenContents/src/presentation/components/tokenContentsCard.tsx
import * as React from "react";
import {
  FileText,
  Upload,
  ChevronLeft,
  ChevronRight,
  Trash2,
} from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

import { MOCK_TOKEN_CONTENTS } from "../../infrastructure/mockdata/mockdata";
import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";
import "../styles/tokenContents.css";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /**
   * 表示するコンテンツ一覧。
   * 未指定の場合は MOCK_TOKEN_CONTENTS を使用。
   */
  contents?: GCSTokenContent[];
  /**
   * 互換用: 画像URL配列として渡された場合も扱えるようにしておく。
   * 新規実装では contents の利用を推奨。
   */
  images?: string[];
  /** 表示モード（edit: 追加/削除可, view: 閲覧専用）。既定: "edit" */
  mode?: Mode;
};

export default function TokenContentsCard({
  contents,
  images,
  mode = "edit",
}: TokenContentsCardProps) {
  const isEditMode = mode === "edit";

  // 初期リストを props / モックから構築（GCSTokenContent[] に正規化）
  const [items, setItems] = React.useState<GCSTokenContent[]>(() => {
    if (contents && contents.length > 0) {
      return contents;
    }
    if (images) {
      return images.map((url, i) => ({
        id: `image_${i + 1}`,
        name: `image_${i + 1}`,
        type: "image",
        url,
        size: 0,
      }));
    }
    return MOCK_TOKEN_CONTENTS;
  });

  const [index, setIndex] = React.useState(0);

  // 外部 props の変化に追従
  React.useEffect(() => {
    if (contents && contents.length > 0) {
      setItems(contents);
      setIndex(0);
      return;
    }
    if (images) {
      const mapped = images.map((url, i) => ({
        id: `image_${i + 1}`,
        name: `image_${i + 1}`,
        type: "image" as const,
        url,
        size: 0,
      }));
      setItems(mapped);
      setIndex(0);
      return;
    }
    setItems(MOCK_TOKEN_CONTENTS);
    setIndex(0);
  }, [contents, images]);

  const hasImages = items.length > 0;

  const prev = () => {
    if (!hasImages) return;
    setIndex((i) => (i - 1 + items.length) % items.length);
  };

  const next = () => {
    if (!hasImages) return;
    setIndex((i) => (i + 1) % items.length);
  };

  const handleUpload = () => {
    if (!isEditMode) return;
    alert("コンテンツファイルの追加（モック）");
  };

  const handleDelete = (targetIndex: number) => {
    if (!isEditMode) return;
    setItems((prev) => {
      if (prev.length === 0) return prev;

      const nextItems = prev.filter((_, i) => i !== targetIndex);

      if (nextItems.length === 0) {
        setIndex(0);
      } else if (targetIndex === index || targetIndex < index) {
        const newIndex = Math.max(0, index - 1);
        setIndex(Math.min(newIndex, nextItems.length - 1));
      } else if (index >= nextItems.length) {
        setIndex(nextItems.length - 1);
      }

      return nextItems;
    });
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

        {/* 編集モード時のみ「ファイル追加」を表示 */}
        {isEditMode && (
          <Button
            type="button"
            className="token-contents-card__add-btn"
            onClick={handleUpload}
          >
            <Upload className="token-contents-card__add-btn-icon" />
            ファイル追加
          </Button>
        )}
      </CardHeader>

      <CardContent>
        {/* メイン画像カルーセル */}
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
              <div className="token-contents-card__image-main-wrap">
                <img
                  src={items[index].url}
                  alt={items[index].name || `コンテンツ ${index + 1}`}
                  className="token-contents-card__image"
                />
                {/* 編集モード時のみメイン画像に削除アイコン表示 */}
                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__delete-btn"
                    onClick={() => handleDelete(index)}
                    aria-label="このコンテンツを削除"
                  >
                    <Trash2 className="token-contents-card__delete-icon" />
                  </button>
                )}
              </div>
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

        {/* サムネイル一覧 */}
        {hasImages && items.length > 1 && (
          <div className="token-contents-card__thumbs">
            {items.map((item, i) => (
              <div
                key={`${item.id}-${i}`}
                className={`token-contents-card__thumb-wrap${
                  i === index ? " is-active" : ""
                }`}
              >
                <button
                  type="button"
                  className="token-contents-card__thumb-click"
                  onClick={() => setIndex(i)}
                  aria-label={`コンテンツ ${i + 1} を表示`}
                >
                  <img
                    src={item.url}
                    alt={item.name || `コンテンツ サムネイル ${i + 1}`}
                    className="token-contents-card__thumb-image"
                  />
                </button>

                {/* 編集モード時のみサムネイルにも削除アイコン表示 */}
                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__thumb-delete-btn"
                    onClick={() => handleDelete(i)}
                    aria-label={`コンテンツ ${i + 1} を削除`}
                  >
                    <Trash2 className="token-contents-card__thumb-delete-icon" />
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

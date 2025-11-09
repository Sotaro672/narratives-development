// frontend/tokenContents/src/pages/tokenContentsCard.tsx
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
} from "../../../shared/ui/card";
import { Button } from "../../../shared/ui/button";

import { MOCK_IMAGES } from "../../mockdata";
import "./tokenContentsCard.css";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /** 初期表示する画像リスト。指定がなければモック、[]なら空表示。 */
  images?: string[];
  /** 表示モード（edit: 追加/削除可, view: 閲覧専用）。既定: "edit" */
  mode?: Mode;
};

export default function TokenContentsCard({
  images,
  mode = "edit",
}: TokenContentsCardProps) {
  const isEditMode = mode === "edit";

  const useExternal = images !== undefined;
  const initialList = useExternal ? images! : MOCK_IMAGES;

  // 表示用リストを state 管理（編集モードで削除できるように）
  const [items, setItems] = React.useState<string[]>(initialList);
  const [index, setIndex] = React.useState(0);

  // 外部 props / モックが変わった場合に同期
  React.useEffect(() => {
    const next = useExternal ? images! : MOCK_IMAGES;
    setItems(next);
    setIndex(0);
  }, [useExternal, images]);

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
      const next = prev.filter((_, i) => i !== targetIndex);
      if (next.length === 0) {
        setIndex(0);
      } else if (targetIndex === index || targetIndex < index) {
        const newIndex = Math.max(0, index - 1);
        setIndex(Math.min(newIndex, next.length - 1));
      } else if (index >= next.length) {
        setIndex(next.length - 1);
      }
      return next;
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
                  src={items[index]}
                  alt={`コンテンツ ${index + 1}`}
                  className="token-contents-card__image"
                />
                {/* 編集モード時のみメイン画像に削除アイコン表示 */}
                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__delete-btn"
                    onClick={() => handleDelete(index)}
                    aria-label="この画像を削除"
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

        {/* サムネイル一覧（2枚目以降を1/9サイズで下に並べる） */}
        {hasImages && items.length > 1 && (
          <div className="token-contents-card__thumbs">
            {items.map((src, i) => (
              <div
                key={`${src}-${i}`}
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
                    src={src}
                    alt={`コンテンツ サムネイル ${i + 1}`}
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

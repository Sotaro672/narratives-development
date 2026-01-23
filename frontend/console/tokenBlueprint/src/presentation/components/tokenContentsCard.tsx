// frontend/console/tokenBlueprint/src/presentation/components/tokenContentsCard.tsx
import * as React from "react";
import { FileText, Upload, ChevronLeft, ChevronRight, Trash2 } from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";
import "../styles/tokenBlueprint.css";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /**
   * 表示するコンテンツ一覧。
   * 未指定の場合は空表示。
   */
  contents?: GCSTokenContent[];

  /** 表示モード（edit: 追加/削除可, view: 閲覧専用）。既定: "edit" */
  mode?: Mode;

  /**
   * file picker でファイルが選択されたときに呼ばれる（推奨）。
   * 実際のアップロード（署名付きURL取得→PUT→contentFiles更新）に接続するための口。
   * 未指定でもローカルプレビュー追加までは動作する。
   */
  onFilesSelected?: (files: File[]) => void | Promise<void>;

  /**
   * edit モードで削除したい時のハンドラ（任意）。
   * 未指定なら UI 内で items を削除するだけ（サーバ反映は呼び出し側で実装）。
   */
  onDelete?: (item: GCSTokenContent, index: number) => void | Promise<void>;
};

function guessContentType(file: File): GCSTokenContent["type"] {
  const mime = String(file.type || "").toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
  return "document";
}

export default function TokenContentsCard({
  contents,
  mode = "edit",
  onFilesSelected,
  onDelete,
}: TokenContentsCardProps) {
  const isEditMode = mode === "edit";

  // props から表示用 items を構築
  const derivedItems = React.useMemo<GCSTokenContent[]>(() => {
    if (contents && contents.length > 0) return contents;
    return [];
  }, [contents]);

  // UI 内での削除（onDelete 未指定でも動作するようにローカル state を持つ）
  const [items, setItems] = React.useState<GCSTokenContent[]>(derivedItems);
  const [index, setIndex] = React.useState(0);

  // file picker
  const inputRef = React.useRef<HTMLInputElement | null>(null);
  const objectUrlsRef = React.useRef<Set<string>>(new Set());

  // 外部 props の変化に追従（※ローカルで追加した分は、呼び出し側で contents を更新したら同期される想定）
  React.useEffect(() => {
    setItems(derivedItems);
    setIndex(0);
  }, [derivedItems]);

  // object URL の後始末
  React.useEffect(() => {
    return () => {
      for (const u of objectUrlsRef.current) {
        try {
          URL.revokeObjectURL(u);
        } catch {
          // noop
        }
      }
      objectUrlsRef.current.clear();
    };
  }, []);

  const hasItems = items.length > 0;

  const prev = () => {
    if (!hasItems) return;
    setIndex((i) => (i - 1 + items.length) % items.length);
  };

  const next = () => {
    if (!hasItems) return;
    setIndex((i) => (i + 1) % items.length);
  };

  const openFilePicker = () => {
    inputRef.current?.click();
  };

  const handleUploadClick = () => {
    if (!isEditMode) return;
    openFilePicker();
  };

  const handleFilesChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (!isEditMode) return;

    const list = e.target.files;
    if (!list || list.length === 0) return;

    const files = Array.from(list);

    // 1) 呼び出し側に通知（任意）
    if (onFilesSelected) {
      await onFilesSelected(files);
    }

    // 2) UI でプレビューできるようにローカルに追加（実データ連携前でも動く）
    const now = Date.now();
    const newItems: GCSTokenContent[] = files.map((f, i) => {
      const url = URL.createObjectURL(f);
      objectUrlsRef.current.add(url);

      const id = `local_${now}_${i}`;
      const name = f.name || id;

      return {
        id,
        name,
        type: guessContentType(f),
        url,
        size: typeof f.size === "number" ? f.size : 0,
      };
    });

    setItems((prevItems) => {
      const merged = [...prevItems, ...newItems];
      if (merged.length > 0) {
        setIndex(Math.max(0, merged.length - newItems.length));
      }
      return merged;
    });

    // 同じファイルを連続で選べるように value をクリア
    e.target.value = "";
  };

  const handleDelete = async (targetIndex: number) => {
    if (!isEditMode) return;

    const target = items[targetIndex];
    if (!target) return;

    // 1) 呼び出し側に通知（任意）
    if (onDelete) {
      await onDelete(target, targetIndex);
    }

    // 2) objectURL なら破棄
    if (typeof target.url === "string" && target.url.startsWith("blob:")) {
      try {
        URL.revokeObjectURL(target.url);
      } catch {
        // noop
      }
      objectUrlsRef.current.delete(target.url);
    }

    // 3) UI から除去（ローカル）
    setItems((prevItems) => {
      if (prevItems.length === 0) return prevItems;

      const nextItems = prevItems.filter((_, i) => i !== targetIndex);

      // index 調整
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
          <CardTitle className="token-contents-card__title">コンテンツ</CardTitle>
        </div>

        {/* hidden file input */}
        <input
          ref={inputRef}
          type="file"
          multiple
          // 必要なら絞る（例: "image/*,video/*,application/pdf"）
          // accept="image/*,video/*,application/pdf"
          style={{ display: "none" }}
          onChange={(e) => void handleFilesChange(e)}
        />

        {/* 編集モード時のみ「ファイル追加」を表示 */}
        {isEditMode && (
          <Button
            type="button"
            className="token-contents-card__add-btn"
            onClick={handleUploadClick}
          >
            <Upload className="token-contents-card__add-btn-icon" />
            ファイル追加
          </Button>
        )}
      </CardHeader>

      <CardContent>
        {/* メイン表示（カルーセル） */}
        <div className="token-contents-card__viewer">
          {/* 左矢印 */}
          <button
            type="button"
            className="token-contents-card__nav token-contents-card__nav--left"
            onClick={prev}
            aria-label="前のコンテンツ"
            disabled={!hasItems}
          >
            <ChevronLeft className="token-contents-card__nav-icon" />
          </button>

          {/* 中央スロット */}
          <div className="token-contents-card__image-slot">
            {hasItems ? (
              <div className="token-contents-card__image-main-wrap">
                <img
                  src={items[index].url}
                  alt={items[index].name || `コンテンツ ${index + 1}`}
                  className="token-contents-card__image"
                />

                {/* 編集モード時のみ削除 */}
                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__delete-btn"
                    onClick={() => void handleDelete(index)}
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
            disabled={!hasItems}
          >
            <ChevronRight className="token-contents-card__nav-icon" />
          </button>
        </div>

        {/* サムネイル一覧 */}
        {hasItems && items.length > 1 && (
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

                {/* 編集モード時のみ削除 */}
                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__thumb-delete-btn"
                    onClick={() => void handleDelete(i)}
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

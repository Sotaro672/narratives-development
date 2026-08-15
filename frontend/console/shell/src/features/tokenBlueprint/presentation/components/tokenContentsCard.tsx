// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenContentsCard.tsx

import * as React from "react";
import { ChevronLeft, ChevronRight, FileText, Trash2, Upload } from "lucide-react";

import { Button } from "../../../../shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import type { ContentFile } from "../../../../shared/types/tokenBlueprint";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /**
   * 表示するコンテンツ一覧。
   * コンテンツの状態管理は親コンポーネントで行う。
   */
  contents?: ContentFile[];

  /**
   * edit:
   * - ファイル追加可能
   * - コンテンツ削除可能
   *
   * view:
   * - 閲覧専用
   */
  mode?: Mode;

  /**
   * file pickerでファイルが選択されたときに呼ばれる。
   * プレビュー生成、Firebase Storage upload、contentFiles保存は呼び出し側で行う。
   */
  onFilesSelected?: (files: File[]) => void | Promise<void>;

  /**
   * editモードでコンテンツを削除するときに呼ばれる。
   * Firebase Storageやbackendへの反映は呼び出し側で行う。
   */
  onDelete?: (item: ContentFile, index: number) => void | Promise<void>;
};

function renderMain(item: ContentFile) {
  switch (item.type) {
    case "image":
      return (
        <img
          src={item.url}
          alt={item.name}
          className="token-contents-card__image"
          onError={(event) => {
            event.currentTarget.style.display = "none";
          }}
        />
      );

    case "video":
      return (
        <video
          className="token-contents-card__video"
          controls
          preload="metadata"
          playsInline
          controlsList="nodownload"
          crossOrigin="anonymous"
        >
          <source src={item.url} type={item.contentType} />
          お使いのブラウザは動画再生に対応していません。
        </video>
      );

    case "pdf":
      return (
        <a
          className="token-contents-card__file-link"
          href={item.url}
          target="_blank"
          rel="noreferrer"
        >
          PDFを開く: {item.name}
        </a>
      );

    case "document":
      return (
        <a
          className="token-contents-card__file-link"
          href={item.url}
          target="_blank"
          rel="noreferrer"
        >
          ファイルを開く: {item.name}
        </a>
      );
  }
}

export default function TokenContentsCard({
  contents = [],
  mode = "edit",
  onFilesSelected,
  onDelete,
}: TokenContentsCardProps) {
  const isEditMode = mode === "edit";
  const [index, setIndex] = React.useState(0);
  const inputRef = React.useRef<HTMLInputElement | null>(null);

  const hasItems = contents.length > 0;
  const safeIndex = React.useMemo(() => {
    if (contents.length === 0) return 0;
    return Math.min(index, contents.length - 1);
  }, [index, contents.length]);

  const currentItem = hasItems ? contents[safeIndex] : undefined;

  React.useEffect(() => {
    setIndex((currentIndex) => {
      if (contents.length === 0) return 0;
      return Math.min(currentIndex, contents.length - 1);
    });
  }, [contents.length]);

  const prev = () => {
    if (!hasItems) return;
    setIndex((currentIndex) => (currentIndex - 1 + contents.length) % contents.length);
  };

  const next = () => {
    if (!hasItems) return;
    setIndex((currentIndex) => (currentIndex + 1) % contents.length);
  };

  const handleUploadClick = () => {
    if (!isEditMode) return;
    inputRef.current?.click();
  };

  const handleFilesChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ): Promise<void> => {
    if (!isEditMode) {
      event.target.value = "";
      return;
    }

    const files = event.target.files ? Array.from(event.target.files) : [];
    event.target.value = "";

    if (files.length === 0 || !onFilesSelected) return;
    await onFilesSelected(files);
  };

  const handleDelete = async (targetIndex: number): Promise<void> => {
    if (!isEditMode || !onDelete) return;

    const target = contents[targetIndex];
    if (!target) return;

    await onDelete(target, targetIndex);
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

        <input
          ref={inputRef}
          type="file"
          multiple
          style={{ display: "none" }}
          onChange={(event) => {
            void handleFilesChange(event);
          }}
        />

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
        <div className="token-contents-card__viewer">
          <button
            type="button"
            className="token-contents-card__nav token-contents-card__nav--left"
            onClick={prev}
            aria-label="前のコンテンツ"
            disabled={!hasItems}
          >
            <ChevronLeft className="token-contents-card__nav-icon" />
          </button>

          <div className="token-contents-card__image-slot">
            {currentItem ? (
              <div className="token-contents-card__image-main-wrap">
                {renderMain(currentItem)}

                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__delete-btn"
                    onClick={() => {
                      void handleDelete(safeIndex);
                    }}
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

        {contents.length > 1 && (
          <div className="token-contents-card__thumbs">
            {contents.map((item, itemIndex) => {
              const isActive = itemIndex === safeIndex;

              return (
                <div
                  key={`${item.id}-${itemIndex}`}
                  className={`token-contents-card__thumb-wrap${isActive ? " is-active" : ""}`}
                >
                  <button
                    type="button"
                    className="token-contents-card__thumb-click"
                    onClick={() => setIndex(itemIndex)}
                    aria-label={`コンテンツ ${itemIndex + 1}を表示`}
                  >
                    {item.type === "image" ? (
                      <img
                        src={item.url}
                        alt={`コンテンツ サムネイル ${itemIndex + 1}`}
                        className="token-contents-card__thumb-image"
                      />
                    ) : (
                      <span className="token-contents-card__thumb-nonimage">
                        {item.type.toUpperCase()}
                      </span>
                    )}
                  </button>

                  {isEditMode && (
                    <button
                      type="button"
                      className="token-contents-card__thumb-delete-btn"
                      onClick={() => {
                        void handleDelete(itemIndex);
                      }}
                      aria-label={`コンテンツ ${itemIndex + 1}を削除`}
                    >
                      <Trash2 className="token-contents-card__thumb-delete-icon" />
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
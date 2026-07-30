// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenContentsCard.tsx

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
} from "../../../../shared/ui/card";
import { Button } from "../../../../shared/ui/button";

import type {
  FirebaseStorageTokenContent,
} from "../../../../shared/types/tokenBlueprint";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /**
   * 表示するコンテンツ一覧。
   * 未指定の場合は空表示。
   *
   * コンテンツの状態管理は親コンポーネントで行う。
   */
  contents?: FirebaseStorageTokenContent[];

  /**
   * 表示モード。
   *
   * edit:
   * - ファイル追加可能
   * - コンテンツ削除可能
   *
   * view:
   * - 閲覧専用
   *
   * 既定値はedit。
   */
  mode?: Mode;

  /**
   * file pickerでファイルが選択されたときに呼ばれる。
   *
   * ファイルのプレビュー生成、Firebase Storageへのアップロード、
   * contentFilesへの保存は、すべて呼び出し側で行う。
   */
  onFilesSelected?: (
    files: File[],
  ) => void | Promise<void>;

  /**
   * editモードでコンテンツを削除するときに呼ばれる。
   *
   * 表示一覧からの削除、Firebase Storageやbackendへの反映は
   * すべて呼び出し側で行う。
   */
  onDelete?: (
    item: FirebaseStorageTokenContent,
    index: number,
  ) => void | Promise<void>;
};

function getVideoMimeType(
  item: FirebaseStorageTokenContent,
): string {
  const url =
    item.url.toLowerCase();

  if (
    url.includes(".webm")
  ) {
    return "video/webm";
  }

  if (
    url.includes(".ogg") ||
    url.includes(".ogv")
  ) {
    return "video/ogg";
  }

  return (
    item.contentType ||
    "video/mp4"
  );
}

function getContentLabel(
  item: FirebaseStorageTokenContent,
): string {
  return (
    item.name ||
    item.id ||
    "content"
  );
}

function renderMain(
  item: FirebaseStorageTokenContent,
) {
  const label =
    getContentLabel(
      item,
    );

  switch (item.type) {
    case "image":
      return (
        <img
          src={item.url}
          alt={label}
          className="token-contents-card__image"
          onError={(event) => {
            event.currentTarget.style.display =
              "none";
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
          <source
            src={item.url}
            type={getVideoMimeType(
              item,
            )}
          />

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
          PDFを開く: {label}
        </a>
      );

    default:
      return (
        <a
          className="token-contents-card__file-link"
          href={item.url}
          target="_blank"
          rel="noreferrer"
        >
          ファイルを開く: {label}
        </a>
      );
  }
}

export default function TokenContentsCard({
  contents,
  mode = "edit",
  onFilesSelected,
  onDelete,
}: TokenContentsCardProps) {
  const isEditMode =
    mode === "edit";

  /**
   * コンテンツの唯一の入力元は、
   * 親コンポーネントから渡されるcontentsとする。
   */
  const items =
    contents ?? [];

  const [
    index,
    setIndex,
  ] = React.useState<number>(
    0,
  );

  const inputRef =
    React.useRef<HTMLInputElement | null>(
      null,
    );

  const hasItems =
    items.length > 0;

  const safeIndex =
    React.useMemo(() => {
      if (
        items.length === 0
      ) {
        return 0;
      }

      return Math.min(
        index,
        items.length - 1,
      );
    }, [
      index,
      items.length,
    ]);

  const currentItem =
    hasItems
      ? items[safeIndex]
      : undefined;

  /**
   * 親コンポーネント側でコンテンツが削除された場合に、
   * 現在位置が配列範囲外にならないよう補正する。
   */
  React.useEffect(() => {
    setIndex(
      (currentIndex) => {
        if (
          items.length === 0
        ) {
          return 0;
        }

        return Math.min(
          currentIndex,
          items.length - 1,
        );
      },
    );
  }, [items.length]);

  const prev = () => {
    if (!hasItems) {
      return;
    }

    setIndex(
      (currentIndex) => {
        return (
          currentIndex -
          1 +
          items.length
        ) % items.length;
      },
    );
  };

  const next = () => {
    if (!hasItems) {
      return;
    }

    setIndex(
      (currentIndex) => {
        return (
          currentIndex +
          1
        ) % items.length;
      },
    );
  };

  const openFilePicker =
    () => {
      inputRef.current?.click();
    };

  const handleUploadClick =
    () => {
      if (!isEditMode) {
        return;
      }

      openFilePicker();
    };

  /**
   * 選択されたファイルを親コンポーネントへ渡す。
   *
   * このコンポーネント内では、
   * Blob URLの作成、ローカルstateへの追加、
   * Firebase Storageへのアップロードは行わない。
   */
  const handleFilesChange =
    async (
      event:
        React.ChangeEvent<HTMLInputElement>,
    ): Promise<void> => {
      if (!isEditMode) {
        event.target.value =
          "";

        return;
      }

      const fileList =
        event.target.files;

      if (
        !fileList ||
        fileList.length === 0
      ) {
        event.target.value =
          "";

        return;
      }

      const files =
        Array.from(
          fileList,
        );

      if (!onFilesSelected) {
        event.target.value =
          "";

        return;
      }

      try {
        await onFilesSelected(
          files,
        );
      } finally {
        event.target.value =
          "";
      }
    };

  /**
   * 削除対象を親コンポーネントへ通知する。
   *
   * このコンポーネント内では、
   * contentsの削除やBlob URLの解放は行わない。
   */
  const handleDelete =
    async (
      targetIndex: number,
    ): Promise<void> => {
      if (!isEditMode) {
        return;
      }

      const target =
        items[targetIndex];

      if (!target) {
        return;
      }

      if (!onDelete) {
        return;
      }

      await onDelete(
        target,
        targetIndex,
      );
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
          style={{
            display: "none",
          }}
          onChange={(event) => {
            void handleFilesChange(
              event,
            );
          }}
        />

        {isEditMode && (
          <Button
            type="button"
            className="token-contents-card__add-btn"
            onClick={
              handleUploadClick
            }
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
            {hasItems &&
            currentItem ? (
              <div className="token-contents-card__image-main-wrap">
                {renderMain(
                  currentItem,
                )}

                {isEditMode && (
                  <button
                    type="button"
                    className="token-contents-card__delete-btn"
                    onClick={() => {
                      void handleDelete(
                        safeIndex,
                      );
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

        {hasItems &&
          items.length > 1 && (
            <div className="token-contents-card__thumbs">
              {items.map(
                (
                  item,
                  itemIndex,
                ) => {
                  const isActive =
                    itemIndex ===
                    safeIndex;

                  return (
                    <div
                      key={`${item.id}-${itemIndex}`}
                      className={
                        `token-contents-card__thumb-wrap${
                          isActive
                            ? " is-active"
                            : ""
                        }`
                      }
                    >
                      <button
                        type="button"
                        className="token-contents-card__thumb-click"
                        onClick={() => {
                          setIndex(
                            itemIndex,
                          );
                        }}
                        aria-label={`コンテンツ ${
                          itemIndex +
                          1
                        }を表示`}
                      >
                        {item.type ===
                        "image" ? (
                          <img
                            src={
                              item.url
                            }
                            alt={`コンテンツ サムネイル ${
                              itemIndex +
                              1
                            }`}
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
                            void handleDelete(
                              itemIndex,
                            );
                          }}
                          aria-label={`コンテンツ ${
                            itemIndex +
                            1
                          }を削除`}
                        >
                          <Trash2 className="token-contents-card__thumb-delete-icon" />
                        </button>
                      )}
                    </div>
                  );
                },
              )}
            </div>
          )}
      </CardContent>
    </Card>
  );
}
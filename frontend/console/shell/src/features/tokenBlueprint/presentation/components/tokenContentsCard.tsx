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

import {
  guessTokenBlueprintContentType,
} from "../../../../shared/types/tokenBlueprint";
import type {
  FirebaseStorageTokenContent,
} from "../../../../shared/types/tokenBlueprint";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /**
   * 表示するコンテンツ一覧。
   * 未指定の場合は空表示。
   */
  contents?: FirebaseStorageTokenContent[];

  /**
   * 表示モード。
   * edit: 追加・削除可能
   * view: 閲覧専用
   *
   * 既定値はedit。
   */
  mode?: Mode;

  /**
   * file pickerでファイルが選択されたときに呼ばれる。
   *
   * 呼び出し側でFirebase Storageへ直接アップロードし、
   * downloadURLとobjectPathをcontentFilesに保存する。
   */
  onFilesSelected?: (files: File[]) => void | Promise<void>;

  /**
   * editモードでコンテンツを削除するときに呼ばれる。
   *
   * backendへの反映は呼び出し側で実装する。
   */
  onDelete?: (
    item: FirebaseStorageTokenContent,
    index: number,
  ) => void | Promise<void>;
};

function nowIso(): string {
  return new Date().toISOString();
}

function buildLocalContent(
  file: File,
  index: number,
  url: string,
  createdAt: string,
): FirebaseStorageTokenContent {
  const timestamp = Date.now();

  return {
    id: `local_${timestamp}_${index}`,
    name: file.name || `local_${index}`,
    type: guessTokenBlueprintContentType(file),
    contentType:
      file.type || "application/octet-stream",
    url,
    objectPath: `local/${timestamp}_${index}`,
    isPublic: false,
    size: file.size,
    createdAt,
    createdBy: "local",
    updatedAt: createdAt,
    updatedBy: "local",
  };
}

function getVideoMimeType(
  item: FirebaseStorageTokenContent,
): string {
  const url = item.url.toLowerCase();

  if (url.includes(".webm")) {
    return "video/webm";
  }

  if (
    url.includes(".ogg") ||
    url.includes(".ogv")
  ) {
    return "video/ogg";
  }

  return item.contentType || "video/mp4";
}

function getContentLabel(
  item: FirebaseStorageTokenContent,
): string {
  return item.name || item.id || "content";
}

function renderMain(
  item: FirebaseStorageTokenContent,
) {
  const label = getContentLabel(item);

  switch (item.type) {
    case "image":
      return (
        <img
          src={item.url}
          alt={label}
          className="token-contents-card__image"
          onError={(event) => {
            // eslint-disable-next-line no-console
            console.warn(
              "[TokenContentsCard] image load failed:",
              item.url,
            );

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
          <source
            src={item.url}
            type={getVideoMimeType(item)}
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
  const isEditMode = mode === "edit";

  const derivedItems = React.useMemo<
    FirebaseStorageTokenContent[]
  >(() => {
    return contents ?? [];
  }, [contents]);

  const [localItems, setLocalItems] = React.useState<
    FirebaseStorageTokenContent[]
  >([]);

  const [index, setIndex] =
    React.useState<number>(0);

  const inputRef =
    React.useRef<HTMLInputElement | null>(null);

  const objectUrlsRef = React.useRef<Set<string>>(
    new Set<string>(),
  );

  /**
   * viewモードでは保存済みコンテンツだけを表示する。
   *
   * editモードでは、保存済みコンテンツの後ろに
   * 今回追加したローカルコンテンツを結合して表示する。
   */
  const items = React.useMemo<
    FirebaseStorageTokenContent[]
  >(() => {
    if (!isEditMode) {
      return derivedItems;
    }

    return [
      ...derivedItems,
      ...localItems,
    ];
  }, [
    derivedItems,
    localItems,
    isEditMode,
  ]);

  const hasItems = items.length > 0;

  const safeIndex = React.useMemo(() => {
    if (items.length === 0) {
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

  const currentItem = hasItems
    ? items[safeIndex]
    : undefined;

  /**
   * コンテンツ数が減少した場合に、
   * 現在位置が配列範囲外にならないよう補正する。
   */
  React.useEffect(() => {
    setIndex((current) => {
      if (items.length === 0) {
        return 0;
      }

      return Math.min(
        current,
        items.length - 1,
      );
    });
  }, [items.length]);

  /**
   * 保存またはキャンセルによってviewモードへ戻ったときに、
   * ローカルプレビューを破棄する。
   *
   * editモード中は、保存済みコンテンツが存在していても
   * localItemsを削除しない。
   */
  React.useEffect(() => {
    if (isEditMode) {
      return;
    }

    for (
      const objectUrl of objectUrlsRef.current
    ) {
      URL.revokeObjectURL(objectUrl);
    }

    objectUrlsRef.current.clear();
    setLocalItems([]);
    setIndex(0);
  }, [isEditMode]);

  /**
   * コンポーネント破棄時にBlob URLを解放する。
   */
  React.useEffect(() => {
    return () => {
      for (
        const objectUrl of objectUrlsRef.current
      ) {
        URL.revokeObjectURL(objectUrl);
      }

      objectUrlsRef.current.clear();
    };
  }, []);

  const prev = () => {
    if (!hasItems) {
      return;
    }

    setIndex((current) => {
      return (
        current -
        1 +
        items.length
      ) % items.length;
    });
  };

  const next = () => {
    if (!hasItems) {
      return;
    }

    setIndex((current) => {
      return (
        current +
        1
      ) % items.length;
    });
  };

  const openFilePicker = () => {
    inputRef.current?.click();
  };

  const handleUploadClick = () => {
    if (!isEditMode) {
      return;
    }

    openFilePicker();
  };

  const handleFilesChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    if (!isEditMode) {
      event.target.value = "";
      return;
    }

    const fileList = event.target.files;

    if (
      !fileList ||
      fileList.length === 0
    ) {
      event.target.value = "";
      return;
    }

    const files = Array.from(fileList);

    if (!onFilesSelected) {
      // eslint-disable-next-line no-console
      console.warn(
        "[TokenContentsCard] onFilesSelected is not provided. No request will be sent to backend.",
      );

      event.target.value = "";
      return;
    }

    try {
      await onFilesSelected(files);
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error(
        "[TokenContentsCard] onFilesSelected failed",
        error,
      );

      event.target.value = "";
      return;
    }

    const createdAt = nowIso();

    const newItems: FirebaseStorageTokenContent[] =
      files.map((file, fileIndex) => {
        const url =
          URL.createObjectURL(file);

        objectUrlsRef.current.add(url);

        return buildLocalContent(
          file,
          fileIndex,
          url,
          createdAt,
        );
      });

    setLocalItems((previousItems) => {
      const nextItems = [
        ...previousItems,
        ...newItems,
      ];

      /**
       * 保存済みコンテンツ数と既存ローカル数の後ろ、
       * つまり今回追加した最初のファイルへ移動する。
       */
      setIndex(
        derivedItems.length +
          previousItems.length,
      );

      return nextItems;
    });

    event.target.value = "";
  };

  const handleDelete = async (
    targetIndex: number,
  ) => {
    if (!isEditMode) {
      return;
    }

    const target = items[targetIndex];

    if (!target) {
      return;
    }

    if (onDelete) {
      try {
        await onDelete(
          target,
          targetIndex,
        );
      } catch (error) {
        // eslint-disable-next-line no-console
        console.error(
          "[TokenContentsCard] onDelete failed",
          error,
        );

        return;
      }
    }

    /**
     * 保存前のローカルコンテンツの場合は、
     * このコンポーネント内のstateから削除する。
     */
    if (target.id.startsWith("local_")) {
      if (target.url.startsWith("blob:")) {
        URL.revokeObjectURL(target.url);

        objectUrlsRef.current.delete(
          target.url,
        );
      }

      setLocalItems((previousItems) => {
        return previousItems.filter(
          (item) => item.id !== target.id,
        );
      });

      setIndex((current) => {
        const nextLength =
          items.length - 1;

        if (nextLength <= 0) {
          return 0;
        }

        return Math.min(
          current,
          nextLength - 1,
        );
      });
    }
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
            {hasItems && currentItem ? (
              <div className="token-contents-card__image-main-wrap">
                {renderMain(currentItem)}

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
                (item, itemIndex) => {
                  const isActive =
                    itemIndex === safeIndex;

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
                          itemIndex + 1
                        }を表示`}
                      >
                        {item.type ===
                        "image" ? (
                          <img
                            src={item.url}
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
                            itemIndex + 1
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
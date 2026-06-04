// frontend/console/tokenBlueprint/src/presentation/components/tokenContentsCard.tsx

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

import type { FirebaseStorageTokenContent } from "../../domain/entity/tokenBlueprint";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  /**
   * 表示するコンテンツ一覧。
   * 未指定の場合は空表示。
   */
  contents?: FirebaseStorageTokenContent[];

  /** 表示モード（edit: 追加/削除可, view: 閲覧専用）。既定: "edit" */
  mode?: Mode;

  /**
   * file picker でファイルが選択されたときに呼ばれる。
   * 呼び出し側で Firebase Storage へ直接アップロードし、
   * downloadURL / objectPath を contentFiles に保存する。
   */
  onFilesSelected?: (files: File[]) => void | Promise<void>;

  /**
   * edit モードで削除したい時のハンドラ。
   * backend への反映は呼び出し側で実装する。
   */
  onDelete?: (
    item: FirebaseStorageTokenContent,
    index: number,
  ) => void | Promise<void>;
};

function guessContentType(file: File): FirebaseStorageTokenContent["type"] {
  const mime = file.type.toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime === "application/pdf") return "pdf";
  return "document";
}

function nowIso(): string {
  return new Date().toISOString();
}

function buildLocalContent(
  file: File,
  index: number,
  url: string,
  createdAt: string,
): FirebaseStorageTokenContent {
  return {
    id: `local_${Date.now()}_${index}`,
    name: file.name || `local_${index}`,
    type: guessContentType(file),
    contentType: file.type || "application/octet-stream",
    url,
    objectPath: `local/${Date.now()}_${index}`,
    visibility: "private",
    size: file.size,
    createdAt,
    createdBy: "local",
    updatedAt: createdAt,
    updatedBy: "local",
  };
}

function getVideoMimeType(item: FirebaseStorageTokenContent): string {
  const url = item.url.toLowerCase();

  if (url.includes(".webm")) return "video/webm";

  if (url.includes(".ogg") || url.includes(".ogv")) {
    return "video/ogg";
  }

  return item.contentType || "video/mp4";
}

function getContentLabel(item: FirebaseStorageTokenContent): string {
  return item.name || item.id || "content";
}

function renderMain(item: FirebaseStorageTokenContent) {
  const label = getContentLabel(item);

  switch (item.type) {
    case "image":
      return (
        <img
          src={item.url}
          alt={label}
          className="token-contents-card__image"
          onError={(e) => {
            // eslint-disable-next-line no-console
            console.warn("[TokenContentsCard] image load failed:", item.url);
            e.currentTarget.style.display = "none";
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
          <source src={item.url} type={getVideoMimeType(item)} />
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
          PDF を開く: {label}
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

  const derivedItems = React.useMemo<FirebaseStorageTokenContent[]>(() => {
    return contents ?? [];
  }, [contents]);

  const [localItems, setLocalItems] = React.useState<
    FirebaseStorageTokenContent[]
  >([]);
  const [index, setIndex] = React.useState(0);

  const inputRef = React.useRef<HTMLInputElement | null>(null);
  const objectUrlsRef = React.useRef<Set<string>>(new Set());

  const items = React.useMemo<FirebaseStorageTokenContent[]>(() => {
    return derivedItems.length > 0 ? derivedItems : localItems;
  }, [derivedItems, localItems]);

  const hasItems = items.length > 0;

  const safeIndex = React.useMemo(() => {
    if (items.length === 0) return 0;
    return Math.min(index, items.length - 1);
  }, [index, items.length]);

  const currentItem = hasItems ? items[safeIndex] : undefined;

  React.useEffect(() => {
    setIndex((current) => {
      if (items.length === 0) return 0;
      return Math.min(current, items.length - 1);
    });
  }, [items.length]);

  React.useEffect(() => {
    if (derivedItems.length > 0) {
      for (const u of objectUrlsRef.current) {
        URL.revokeObjectURL(u);
      }

      objectUrlsRef.current.clear();
      setLocalItems([]);
      setIndex(0);
      return;
    }

    setIndex((i) => {
      const len = items.length;
      if (len === 0) return 0;
      return Math.min(i, len - 1);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [derivedItems.length]);

  React.useEffect(() => {
    return () => {
      for (const u of objectUrlsRef.current) {
        URL.revokeObjectURL(u);
      }

      objectUrlsRef.current.clear();
    };
  }, []);

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

    if (!onFilesSelected) {
      // eslint-disable-next-line no-console
      console.warn(
        "[TokenContentsCard] onFilesSelected is not provided. No request will be sent to backend.",
      );
      e.target.value = "";
      return;
    }

    try {
      await onFilesSelected(files);
    } catch (err) {
      // eslint-disable-next-line no-console
      console.error("[TokenContentsCard] onFilesSelected failed", err);
      e.target.value = "";
      return;
    }

    const createdAt = nowIso();
    const newItems: FirebaseStorageTokenContent[] = files.map((f, i) => {
      const url = URL.createObjectURL(f);
      objectUrlsRef.current.add(url);

      return buildLocalContent(f, i, url, createdAt);
    });

    setLocalItems((prevItems) => {
      const merged = [...prevItems, ...newItems];

      if (merged.length > 0) {
        setIndex(Math.max(0, merged.length - newItems.length));
      }

      return merged;
    });

    e.target.value = "";
  };

  const handleDelete = async (targetIndex: number) => {
    if (!isEditMode) return;

    const target = items[targetIndex];
    if (!target) return;

    if (onDelete) {
      try {
        await onDelete(target, targetIndex);
      } catch (err) {
        // eslint-disable-next-line no-console
        console.error("[TokenContentsCard] onDelete failed", err);
        return;
      }
    }

    if (target.id.startsWith("local_")) {
      if (target.url.startsWith("blob:")) {
        URL.revokeObjectURL(target.url);
        objectUrlsRef.current.delete(target.url);
      }

      setLocalItems((prevItems) => {
        return prevItems.filter((x) => x.id !== target.id);
      });

      setIndex((i) => {
        const len = items.length - 1;
        if (len <= 0) return 0;
        return Math.min(i, len - 1);
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
          style={{ display: "none" }}
          onChange={(e) => void handleFilesChange(e)}
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
                    onClick={() => void handleDelete(safeIndex)}
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

        {hasItems && items.length > 1 && (
          <div className="token-contents-card__thumbs">
            {items.map((item, i) => (
              <div
                key={`${item.id}-${i}`}
                className={`token-contents-card__thumb-wrap${
                  i === safeIndex ? " is-active" : ""
                }`}
              >
                <button
                  type="button"
                  className="token-contents-card__thumb-click"
                  onClick={() => setIndex(i)}
                  aria-label={`コンテンツ ${i + 1} を表示`}
                >
                  {item.type === "image" ? (
                    <img
                      src={item.url}
                      alt={`コンテンツ サムネイル ${i + 1}`}
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
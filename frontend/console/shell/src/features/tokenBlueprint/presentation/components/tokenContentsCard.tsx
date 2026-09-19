// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenContentsCard.tsx

import * as React from "react";
import {
  ChevronLeft,
  ChevronRight,
  FileText,
  Trash2,
  Upload,
} from "lucide-react";

import { IMAGE_STORAGE_ACCEPT } from "../../../../shared/storage/imageStoragePolicy";
import type { ContentFile } from "../../../../shared/types/tokenBlueprint";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardButton,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import { Media } from "../../../../shared/ui/media";

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

function ContentMainMedia({
  item,
}: {
  item?: ContentFile;
}) {
  if (!item) {
    return (
      <Media
        variant="viewer"
        fit="contain"
        emptyIcon={<FileText />}
        emptyText="コンテンツがまだ登録されていません"
      />
    );
  }

  return (
    <Media
      src={item.url}
      type={item.type}
      name={item.name}
      alt={item.name}
      contentType={item.contentType}
      variant="viewer"
      fit="contain"
      imageProps={{
        onError: (event) => {
          event.currentTarget.style.display = "none";
        },
      }}
      videoProps={{
        controls: true,
        preload: "metadata",
        playsInline: true,
        controlsList: "nodownload",
        crossOrigin: "anonymous",
      }}
    />
  );
}

function ContentThumbnail({
  item,
  index,
  onActivate,
}: {
  item: ContentFile;
  index: number;
  onActivate: () => void;
}) {
  if (item.type === "image") {
    return (
      <Media
        src={item.url}
        type="image"
        alt={`コンテンツ サムネイル ${index + 1}`}
        variant="square"
        fit="cover"
        bordered={false}
        onActivate={onActivate}
      />
    );
  }

  return (
    <Media
      variant="square"
      bordered={false}
      emptyText={item.type.toUpperCase()}
      emptyDescription={item.name}
      onActivate={onActivate}
    />
  );
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
    if (contents.length === 0) {
      return 0;
    }

    return Math.min(index, contents.length - 1);
  }, [index, contents.length]);

  const currentItem = hasItems
    ? contents[safeIndex]
    : undefined;

  React.useEffect(() => {
    setIndex((currentIndex) => {
      if (contents.length === 0) {
        return 0;
      }

      return Math.min(
        currentIndex,
        contents.length - 1,
      );
    });
  }, [contents.length]);

  const prev = () => {
    if (!hasItems) {
      return;
    }

    setIndex(
      (currentIndex) =>
        (currentIndex - 1 + contents.length) %
        contents.length,
    );
  };

  const next = () => {
    if (!hasItems) {
      return;
    }

    setIndex(
      (currentIndex) =>
        (currentIndex + 1) %
        contents.length,
    );
  };

  const handleUploadClick = () => {
    if (!isEditMode) {
      return;
    }

    inputRef.current?.click();
  };

  const handleFilesChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ): Promise<void> => {
    if (!isEditMode) {
      event.target.value = "";
      return;
    }

    const files = event.target.files
      ? Array.from(event.target.files)
      : [];

    event.target.value = "";

    if (files.length === 0 || !onFilesSelected) {
      return;
    }

    await onFilesSelected(files);
  };

  const handleDelete = async (
    targetIndex: number,
  ): Promise<void> => {
    if (!isEditMode || !onDelete) {
      return;
    }

    const target = contents[targetIndex];

    if (!target) {
      return;
    }

    await onDelete(target, targetIndex);
  };

  return (
    <Card elevated largeRadius>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon variant="primary">
            <FileText className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>
            コンテンツ
          </CardTitle>
        </CardHeaderLeft>

        <input
          ref={inputRef}
          type="file"
          accept={IMAGE_STORAGE_ACCEPT}
          multiple
          hidden
          onChange={(event) => {
            void handleFilesChange(event);
          }}
        />

        {isEditMode && (
          <CardButton
            variant="primary"
            onClick={handleUploadClick}
          >
            <Upload className="card__button-icon" />
            ファイル追加
          </CardButton>
        )}
      </CardHeader>

      <CardContent size="large">
        <div className="token-contents-card__viewer">
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="token-contents-card__nav token-contents-card__nav--left"
            onClick={prev}
            aria-label="前のコンテンツ"
            disabled={!hasItems}
          >
            <ChevronLeft className="token-contents-card__nav-icon" />
          </Button>

          <div className="token-contents-card__image-main-wrap">
            <ContentMainMedia item={currentItem} />

            {currentItem && isEditMode && (
              <Button
                type="button"
                variant="destructive-outline"
                size="icon"
                className="token-contents-card__delete-btn"
                onClick={() => {
                  void handleDelete(safeIndex);
                }}
                aria-label="このコンテンツを削除"
                title="削除"
              >
                <Trash2 className="token-contents-card__delete-icon" />
              </Button>
            )}
          </div>

          <Button
            type="button"
            variant="outline"
            size="icon"
            className="token-contents-card__nav token-contents-card__nav--right"
            onClick={next}
            aria-label="次のコンテンツ"
            disabled={!hasItems}
          >
            <ChevronRight className="token-contents-card__nav-icon" />
          </Button>
        </div>

        {contents.length > 1 && (
          <div className="token-contents-card__thumbs">
            {contents.map((item, itemIndex) => {
              const isActive = itemIndex === safeIndex;

              return (
                <div
                  key={`${item.id}-${itemIndex}`}
                  className={`token-contents-card__thumb-wrap${
                    isActive ? " is-active" : ""
                  }`}
                >
                  <ContentThumbnail
                    item={item}
                    index={itemIndex}
                    onActivate={() => {
                      setIndex(itemIndex);
                    }}
                  />

                  {isEditMode && (
                    <Button
                      type="button"
                      variant="destructive-outline"
                      size="icon"
                      className="token-contents-card__thumb-delete-btn"
                      onClick={() => {
                        void handleDelete(itemIndex);
                      }}
                      aria-label={`コンテンツ ${itemIndex + 1}を削除`}
                      title="削除"
                    >
                      <Trash2 className="token-contents-card__thumb-delete-icon" />
                    </Button>
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
// frontend/console/shell/src/features/tokenBlueprint/presentation/components/tokenContentsCard.tsx

import {
  FileText,
  Upload,
} from "lucide-react";

import { IMAGE_STORAGE_ACCEPT } from "../../../../shared/storage/imageStoragePolicy";
import type { ContentFile } from "../../../../shared/types/tokenBlueprint";
import {
  Card,
  CardButton,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import MediaGallery from "../../../../shared/ui/mediaGallery";
import MediaUploader from "../../../../shared/ui/mediaUploader";

type Mode = "edit" | "view";

type TokenContentsCardProps = {
  contents?: ContentFile[];
  mode?: Mode;
  onFilesSelected?: (files: File[]) => void | Promise<void>;
  onDelete?: (
    item: ContentFile,
    index: number,
  ) => void | Promise<void>;
};

export default function TokenContentsCard({
  contents = [],
  mode = "edit",
  onFilesSelected,
  onDelete,
}: TokenContentsCardProps) {
  const isEditMode = mode === "edit";
  const hasItems = contents.length > 0;

  const galleryItems = contents.map((item) => ({
    id: item.id,
    src: item.url,
    name: item.name,
    alt: item.name,
    type: item.type,
    contentType: item.contentType,
  }));

  const handleFilesSelected = (
    files: File[],
  ): void => {
    void onFilesSelected?.(files);
  };

  const handleDelete = (
    itemIndex: number,
  ): void => {
    if (!isEditMode || !onDelete) {
      return;
    }

    const target = contents[itemIndex];

    if (!target) {
      return;
    }

    void onDelete(target, itemIndex);
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

        {isEditMode && hasItems ? (
          <MediaUploader
            accept={IMAGE_STORAGE_ACCEPT}
            multiple
            title={null}
            showCount={false}
            showPicker={false}
            disabled={!onFilesSelected}
            className="token-contents-card__uploader"
            renderEmpty={({ openPicker, disabled }) => (
              <CardButton
                variant="primary"
                disabled={disabled}
                onClick={openPicker}
              >
                <Upload className="card__button-icon" />
                ファイル追加
              </CardButton>
            )}
            onFilesSelected={handleFilesSelected}
          />
        ) : null}
      </CardHeader>

      <CardContent size="large">
        {!hasItems && isEditMode ? (
          <MediaUploader
            accept={IMAGE_STORAGE_ACCEPT}
            multiple
            title={null}
            showCount={false}
            pickerLabel="メディアをアップロード"
            pickerDescription="クリックまたはドラッグ＆ドロップでファイルを追加できます"
            disabled={!onFilesSelected}
            className="token-contents-card__empty-uploader"
            onFilesSelected={handleFilesSelected}
          />
        ) : (
          <MediaGallery
            items={galleryItems}
            editable={isEditMode}
            emptyIcon={<FileText />}
            emptyText="コンテンツがまだ登録されていません"
            mainVariant="viewer"
            mainFit="contain"
            thumbnailFit="cover"
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
            onDelete={
              onDelete
                ? (_item, itemIndex) => {
                    handleDelete(itemIndex);
                  }
                : undefined
            }
          />
        )}
      </CardContent>
    </Card>
  );
}
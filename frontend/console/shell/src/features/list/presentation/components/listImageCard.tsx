// frontend/console/shell/src/features/list/presentation/components/listImageCard.tsx

import {
  Image as ImageIcon,
  Upload,
} from "lucide-react";

import { IMAGE_STORAGE_ACCEPT } from "../../../../shared/storage/imageStoragePolicy";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardButton,
  CardContent,
  CardHeader,
  CardHeaderActions,
  CardHeaderIcon,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import MediaGallery from "../../../../shared/ui/mediaGallery";
import MediaUploader from "../../../../shared/ui/mediaUploader";

import { useListImageCard } from "../hook/useListImageCard";

export type ListImageCardProps = {
  isEdit: boolean;
  saving?: boolean;
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: (idx: number) => void;
  onAddImages?: (files: File[]) => void;
  onRemoveImageAt?: (idx: number) => void;
  onClearImages?: () => void;
};

export default function ListImageCard(
  props: ListImageCardProps,
) {
  const vm = useListImageCard({
    isEdit: props.isEdit,
    imageUrls: props.imageUrls,
    mainImageIndex: props.mainImageIndex,
    setMainImageIndex: props.setMainImageIndex,
    onRemoveImageAt: props.onRemoveImageAt,
    onClearImages: props.onClearImages,
  });

  const canAddImages =
    props.isEdit &&
    !props.saving &&
    Boolean(props.onAddImages);

  const galleryItems =
    vm.effectiveImageUrls.map((url, index) => ({
      id: `${index}-${url}`,
      src: url,
      type: "image" as const,
      alt: `商品画像 ${index + 1}`,
    }));

  const handleFilesSelected = (
    files: File[],
  ): void => {
    if (!canAddImages) {
      return;
    }

    props.onAddImages?.(files);
  };

  return (
    <Card>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <ImageIcon className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>
            商品画像
          </CardTitle>
        </CardHeaderLeft>

        {props.isEdit && vm.hasImages ? (
          <CardHeaderActions>
            <MediaUploader
              accept={IMAGE_STORAGE_ACCEPT}
              multiple
              title={null}
              showCount={false}
              showPicker={false}
              disabled={!canAddImages}
              className="card__header-uploader"
              renderEmpty={({ openPicker, disabled }) => (
                <CardButton
                  variant="primary"
                  disabled={disabled}
                  onClick={openPicker}
                >
                  <Upload className="card__button-icon" />
                  画像追加
                </CardButton>
              )}
              onFilesSelected={handleFilesSelected}
            />

            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={vm.handleClear}
              disabled={Boolean(props.saving)}
            >
              クリア
            </Button>
          </CardHeaderActions>
        ) : null}
      </CardHeader>

      <CardContent>
        {!vm.hasImages && props.isEdit ? (
          <MediaUploader
            accept={IMAGE_STORAGE_ACCEPT}
            multiple
            title={null}
            showCount={false}
            pickerLabel="画像をアップロード"
            pickerDescription="クリックまたはドラッグ＆ドロップで画像を追加できます"
            emptyIcon={<ImageIcon />}
            disabled={!canAddImages}
            onFilesSelected={handleFilesSelected}
          />
        ) : (
          <MediaGallery
            items={galleryItems}
            activeIndex={props.mainImageIndex}
            onActiveIndexChange={vm.handleSetMainIndex}
            editable={props.isEdit}
            deleteDisabled={Boolean(props.saving)}
            mainVariant="viewer"
            mainFit="contain"
            thumbnailFit="cover"
            emptyIcon={<ImageIcon />}
            emptyText="商品画像がまだ登録されていません"
            emptyDescription={
              props.isEdit
                ? "画像を追加してください"
                : "編集モードで追加できます"
            }
            onDelete={
              props.onRemoveImageAt
                ? (_item, itemIndex) => {
                    vm.handleRemoveAt(itemIndex);
                  }
                : undefined
            }
          />
        )}
      </CardContent>
    </Card>
  );
}
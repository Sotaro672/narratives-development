// frontend/console/shell/src/features/list/presentation/components/listImageCard.tsx
// 商品画像カード（表示はshared/ui/media、ファイル選択はshared/ui/mediaUploader、ロジックはhookに委譲）

import { Image as ImageIcon, Plus } from "lucide-react";

import { IMAGE_STORAGE_ACCEPT } from "../../../../shared/storage/imageStoragePolicy";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import DeleteButton from "../../../../shared/ui/delete";
import { Media } from "../../../../shared/ui/media";
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

export default function ListImageCard(props: ListImageCardProps) {
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

  return (
    <Card>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <ImageIcon className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>商品画像</CardTitle>
        </CardHeaderLeft>

        {props.isEdit && vm.effectiveImageUrls.length > 0 && (
          <div className="lic__actions">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={vm.handleClear}
              disabled={Boolean(props.saving)}
            >
              クリア
            </Button>
          </div>
        )}
      </CardHeader>

      <CardContent>
        <MediaUploader
          accept={IMAGE_STORAGE_ACCEPT}
          multiple
          variant="grid"
          title={null}
          showCount={false}
          showPicker={false}
          showFileNames={false}
          showRemoveButton={false}
          previewFramed={false}
          renderEmpty={({ openPicker }) => (
            <>
              {!vm.hasImages && (
                <Media
                  variant="landscape"
                  fit="cover"
                  emptyIcon={<ImageIcon />}
                  emptyText="画像を追加"
                  emptyDescription={
                    props.isEdit
                      ? "クリックで選択（複数可）"
                      : "編集モードで追加できます"
                  }
                  onActivate={
                    canAddImages
                      ? openPicker
                      : undefined
                  }
                />
              )}

              {vm.hasImages && (
                <>
                  <div className="lic__main">
                    <Media
                      src={vm.mainUrl}
                      type="image"
                      alt="商品メイン画像"
                      variant="landscape"
                      fit="cover"
                      bordered={false}
                      onActivate={
                        canAddImages
                          ? openPicker
                          : undefined
                      }
                    />

                    {props.isEdit && (
                      <DeleteButton
                        size="md"
                        className="lic__remove-btn"
                        ariaLabel="メイン画像を削除"
                        disabled={Boolean(props.saving)}
                        onClick={(event) => {
                          event.stopPropagation();
                          vm.handleRemoveAt(props.mainImageIndex);
                        }}
                      />
                    )}

                    <div className="lic__footer">
                      <div className="lic__footer-left">
                        {vm.effectiveImageUrls.length} 枚
                        {props.isEdit
                          ? "（×で削除 / クリックで追加）"
                          : "（サムネでメイン切替）"}
                      </div>

                      {!props.isEdit && (
                        <div className="lic__footer-note">
                          ※ 画像変更は編集モードで行えます
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="lic__grid">
                    {vm.thumbIndices.map((idx: number) => {
                      const url =
                        vm.effectiveImageUrls[idx] ?? "";

                      return (
                        <div
                          key={`${url}-${idx}`}
                          className="lic__thumb"
                        >
                          <Media
                            src={url}
                            type="image"
                            alt={`商品画像 ${idx + 1}`}
                            variant="square"
                            fit="cover"
                            bordered={false}
                            onActivate={() =>
                              vm.handleSetMainIndex(idx)
                            }
                          />

                          {props.isEdit && (
                            <DeleteButton
                              size="sm"
                              className="lic__thumb-remove"
                              ariaLabel={`商品画像 ${idx + 1} を削除`}
                              disabled={Boolean(props.saving)}
                              onClick={(event) => {
                                event.stopPropagation();
                                vm.handleRemoveAt(idx);
                              }}
                            />
                          )}
                        </div>
                      );
                    })}

                    {props.isEdit && (
                      <Media
                        variant="square"
                        emptyIcon={<Plus />}
                        emptyText="画像を追加"
                        onActivate={
                          canAddImages
                            ? openPicker
                            : undefined
                        }
                        disabled={!canAddImages}
                      />
                    )}
                  </div>
                </>
              )}
            </>
          )}
          onFilesSelected={(files) => {
            if (!canAddImages) {
              return;
            }

            props.onAddImages?.(files);
          }}
        />
      </CardContent>
    </Card>
  );
}
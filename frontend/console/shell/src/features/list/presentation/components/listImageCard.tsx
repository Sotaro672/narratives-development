// frontend/console/list/src/presentation/components/listImageCard.tsx
// 商品画像カード（表示はshared/ui/media、ロジックはhookに委譲）

import { Image as ImageIcon, Plus, X } from "lucide-react";

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
import { Media } from "../../../../shared/ui/media";

import { useListImageCard } from "../hook/useListImageCard";

export type ListImageCardProps = {
  isEdit: boolean;
  saving?: boolean;
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: (idx: number) => void;
  onAddImages?: (files: FileList | null) => void;
  onRemoveImageAt?: (idx: number) => void;
  onClearImages?: () => void;
};

export default function ListImageCard(props: ListImageCardProps) {
  const vm = useListImageCard({
    isEdit: props.isEdit,
    imageUrls: props.imageUrls,
    mainImageIndex: props.mainImageIndex,
    setMainImageIndex: props.setMainImageIndex,
    onAddImages: props.onAddImages,
    onRemoveImageAt: props.onRemoveImageAt,
    onClearImages: props.onClearImages,
  });

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
        <input
          ref={vm.imageInputRef as any}
          type="file"
          accept={IMAGE_STORAGE_ACCEPT}
          multiple
          hidden
          onChange={vm.handleInputChange}
        />

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
              props.isEdit
                ? vm.openPicker
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
                  props.isEdit
                    ? vm.openPicker
                    : undefined
                }
              />

              {props.isEdit && (
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="lic__remove-btn"
                  onClick={(event) => {
                    event.stopPropagation();
                    vm.handleRemoveAt(props.mainImageIndex);
                  }}
                  aria-label="メイン画像を削除"
                  title="削除"
                  disabled={Boolean(props.saving)}
                >
                  <X aria-hidden="true" />
                </Button>
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
                const url = vm.effectiveImageUrls[idx] ?? "";

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
                      onActivate={() => vm.handleSetMainIndex(idx)}
                    />

                    {props.isEdit && (
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        className="lic__thumb-remove"
                        onClick={(event) => {
                          event.stopPropagation();
                          vm.handleRemoveAt(idx);
                        }}
                        aria-label={`商品画像 ${idx + 1} を削除`}
                        title="削除"
                        disabled={Boolean(props.saving)}
                      >
                        <X aria-hidden="true" />
                      </Button>
                    )}
                  </div>
                );
              })}

              {props.isEdit && (
                <Media
                  variant="square"
                  emptyIcon={<Plus />}
                  emptyText="画像を追加"
                  onActivate={vm.openPicker}
                />
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
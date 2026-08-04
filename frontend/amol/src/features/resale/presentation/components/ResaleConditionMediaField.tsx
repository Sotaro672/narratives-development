// frontend/amol/src/features/resale/presentation/components/ResaleConditionMediaField.tsx

import type {
  ChangeEvent,
  RefObject,
} from "react";

import MediaUploader from "../../../../components/ui/MediaUploader";

import type {
  MediaUploaderItem,
} from "../../../../components/ui/MediaUploader";

export type ResaleConditionMediaFieldProps = {
  items: MediaUploaderItem[];
  currentIndex: number;
  inputRef: RefObject<HTMLInputElement>;
  carouselRef: RefObject<HTMLDivElement>;
  disabled?: boolean;
  selecting?: boolean;
  onFilesSelected: (
    event: ChangeEvent<HTMLInputElement>,
  ) => void;
  onRemoveItem: (id: string) => void;
  onCarouselScroll: () => void;
  onMoveToSlide: (index: number) => void;
};

export default function ResaleConditionMediaField({
  items,
  currentIndex,
  inputRef,
  carouselRef,
  disabled = false,
  selecting = false,
  onFilesSelected,
  onRemoveItem,
  onCarouselScroll,
  onMoveToSlide,
}: ResaleConditionMediaFieldProps) {
  return (
    <MediaUploader
      label="商品状態の写真"
      hint="傷・汚れ・タグ・付属品など、購入者が状態を確認できる写真を追加してください。必須項目です。"
      emptyText="商品状態の写真が登録されていません。"
      selectButtonLabel="写真を追加"
      selectingButtonLabel="追加中..."
      accept="image/*"
      multiple
      items={items}
      currentIndex={currentIndex}
      inputRef={inputRef}
      carouselRef={carouselRef}
      disabled={disabled}
      selecting={selecting}
      onFilesSelected={onFilesSelected}
      onRemoveItem={onRemoveItem}
      onCarouselScroll={onCarouselScroll}
      onMoveToSlide={onMoveToSlide}
    />
  );
}
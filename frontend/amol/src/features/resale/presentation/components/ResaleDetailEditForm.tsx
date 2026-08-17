// frontend/amol/src/features/resale/presentation/components/ResaleDetailEditForm.tsx

import type { MouseEventHandler } from "react";

import Dropdown from "../../../../components/ui/Dropdown";
import Input from "../../../../components/ui/Input";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Textbox from "../../../../components/ui/Textbox";

import {
  RESALE_CONDITION_OPTIONS,
  RESALE_STATUS_OPTIONS,
} from "../../../shared/types/resale";
import { getResaleEditableStatusLabel } from "../../constants/resaleStatusOptions";

import type { ResaleDetailEditFormProps } from "../types/resaleDetailPageTypes";

import ResaleConditionMediaField from "./ResaleConditionMediaField";

const DESCRIPTION_MAX_LENGTH = 1000;

type DropdownTriggerProps = {
  label: string;
  isOpen: boolean;
  disabled: boolean;
  onClick: MouseEventHandler<HTMLButtonElement>;
};

function DropdownTrigger({
  label,
  isOpen,
  disabled,
  onClick,
}: DropdownTriggerProps) {
  return (
    <button
      type="button"
      className="page-form__dropdown-button"
      disabled={disabled}
      aria-expanded={isOpen}
      onClick={onClick}
    >
      <span>{label}</span>
      <span aria-hidden="true">{isOpen ? "▲" : "▼"}</span>
    </button>
  );
}

export default function ResaleDetailEditForm({
  priceValue,
  condition,
  status,
  description,
  saving,
  createdAtLabel,
  updatedAtLabel,
  conditionMediaItems,
  conditionMediaCurrentIndex,
  conditionMediaInputRef,
  conditionMediaCarouselRef,
  onPriceChange,
  onConditionChange,
  onStatusChange,
  onDescriptionChange,
  onConditionMediaSelected,
  onRemoveConditionMedia,
  onConditionMediaCarouselScroll,
  onMoveToConditionMediaSlide,
}: ResaleDetailEditFormProps) {
  const statusLabel = getResaleEditableStatusLabel(status);

  return (
    <section className="page-card">
      <SectionHeader title="販売情報" titleAs="h2" />

      <div className="page-form">
        <Input
          label="販売価格"
          type="text"
          inputMode="numeric"
          value={priceValue}
          placeholder="例：12,000"
          helperText="半角数字で入力してください。"
          required
          disabled={saving}
          onChange={(event) => {
            onPriceChange(event.currentTarget.value);
          }}
        />

        <div className="page-form__field">
          <span className="page-form__label">商品の状態</span>
          <Dropdown
            buttonLabel={condition}
            items={RESALE_CONDITION_OPTIONS}
            selectedValue={condition}
            onSelect={onConditionChange}
            renderButton={({ isOpen, toggle }) => (
              <DropdownTrigger
                label={condition}
                isOpen={isOpen}
                disabled={saving}
                onClick={toggle}
              />
            )}
          />
        </div>

        <div className="page-form__field">
          <span className="page-form__label">公開状態</span>
          <Dropdown
            buttonLabel={statusLabel}
            items={RESALE_STATUS_OPTIONS}
            selectedValue={status}
            onSelect={onStatusChange}
            renderButton={({ isOpen, toggle }) => (
              <DropdownTrigger
                label={statusLabel}
                isOpen={isOpen}
                disabled={saving}
                onClick={toggle}
              />
            )}
          />
        </div>

        <ResaleConditionMediaField
          items={conditionMediaItems}
          currentIndex={conditionMediaCurrentIndex}
          inputRef={conditionMediaInputRef}
          carouselRef={conditionMediaCarouselRef}
          disabled={saving}
          selecting={saving}
          onFilesSelected={onConditionMediaSelected}
          onRemoveItem={onRemoveConditionMedia}
          onCarouselScroll={onConditionMediaCarouselScroll}
          onMoveToSlide={onMoveToConditionMediaSlide}
        />

        <Textbox
          label="説明文"
          value={description}
          placeholder="購入時期、着用回数、保管状態などを入力してください。"
          rows={6}
          maxLength={DESCRIPTION_MAX_LENGTH}
          helperText="購入者が商品の状態を判断しやすい内容を入力してください。"
          counterText={`${description.length}/${DESCRIPTION_MAX_LENGTH}`}
          disabled={saving}
          onChange={(event) => {
            onDescriptionChange(event.currentTarget.value);
          }}
        />
      </div>

      <dl className="page-definition-list resale-detail-page__readonly-meta">
        <div className="page-definition-list__row">
          <dt>出品日時</dt>
          <dd>{createdAtLabel}</dd>
        </div>
        <div className="page-definition-list__row">
          <dt>更新日時</dt>
          <dd>{updatedAtLabel}</dd>
        </div>
      </dl>
    </section>
  );
}
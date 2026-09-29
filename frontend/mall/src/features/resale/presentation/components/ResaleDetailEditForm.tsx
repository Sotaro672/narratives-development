// frontend/mall/src/features/resale/presentation/components/ResaleDetailEditForm.tsx

import Dropdown from "../../../../components/ui/Dropdown";
import Input from "../../../../components/ui/Input";
import Textbox from "../../../../components/ui/Textbox";

import ProductMetaList from "../../../shared/presentation/components/ProductMetaList";
import {
  RESALE_CONDITION_OPTIONS,
  RESALE_STATUS_OPTIONS,
} from "../../../shared/types/resale";
import { getResaleEditableStatusLabel } from "../../constants/resaleStatusOptions";

import type { ResaleDetailEditFormProps } from "../types/resaleDetailPageTypes";

const DESCRIPTION_MAX_LENGTH = 1000;

export default function ResaleDetailEditForm({
  priceValue,
  condition,
  status,
  description,
  saving,
  createdAtLabel,
  updatedAtLabel,
  onPriceChange,
  onConditionChange,
  onStatusChange,
  onDescriptionChange,
}: ResaleDetailEditFormProps) {
  const statusLabel = getResaleEditableStatusLabel(status);

  return (
    <div className="resale-detail-page__edit-form">
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
          onChange={(event) => onPriceChange(event.currentTarget.value)}
        />

        <div className="page-form__field">
          <span className="page-form__label">商品の状態</span>
          <Dropdown
            buttonLabel={condition}
            items={RESALE_CONDITION_OPTIONS}
            selectedValue={condition}
            onSelect={onConditionChange}
            disabled={saving}
          />
        </div>

        <div className="page-form__field">
          <span className="page-form__label">公開状態</span>
          <Dropdown
            buttonLabel={statusLabel}
            items={RESALE_STATUS_OPTIONS}
            selectedValue={status}
            onSelect={onStatusChange}
            disabled={saving}
          />
        </div>

        <Textbox
          label="説明文"
          value={description}
          placeholder="購入時期、着用回数、保管状態などを入力してください。"
          rows={6}
          maxLength={DESCRIPTION_MAX_LENGTH}
          helperText="購入者が商品の状態を判断しやすい内容を入力してください。"
          counterText={`${description.length}/${DESCRIPTION_MAX_LENGTH}`}
          disabled={saving}
          onChange={(event) => onDescriptionChange(event.currentTarget.value)}
        />
      </div>

      <ProductMetaList
        className="resale-detail-page__edit-meta"
        items={[
          { label: "出品日時", value: createdAtLabel },
          { label: "更新日時", value: updatedAtLabel },
        ]}
      />
    </div>
  );
}
// frontend/amol/src/features/resale/presentation/components/ResaleCreateForm.tsx

import Input from "../../../../components/ui/Input";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Textbox from "../../../../components/ui/Textbox";

import {
  RESALE_CONDITION_OPTIONS,
  type ResaleCondition,
} from "../../../shared/types/resale";

const DESCRIPTION_MAX_LENGTH = 1000;

export type ResaleCreateFormProps = {
  formattedPrice: string;
  condition: ResaleCondition;
  description: string;
  disabled?: boolean;
  onPriceChange: (value: string) => void;
  onConditionChange: (value: ResaleCondition) => void;
  onDescriptionChange: (value: string) => void;
};

export default function ResaleCreateForm({
  formattedPrice,
  condition,
  description,
  disabled = false,
  onPriceChange,
  onConditionChange,
  onDescriptionChange,
}: ResaleCreateFormProps) {
  return (
    <section className="page-card">
      <SectionHeader title="販売情報" titleAs="h2" />

      <div className="page-form">
        <Input
          label="販売価格"
          type="text"
          inputMode="numeric"
          value={formattedPrice}
          placeholder="例：12,000"
          helperText="半角数字で入力してください。"
          required
          disabled={disabled}
          onChange={(event) => onPriceChange(event.currentTarget.value)}
        />

        <label className="page-form__field">
          <span className="page-form__label">商品の状態</span>

          <select
            value={condition}
            disabled={disabled}
            onChange={(event) =>
              onConditionChange(event.currentTarget.value as ResaleCondition)
            }
          >
            {RESALE_CONDITION_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        <Textbox
          label="説明文"
          value={description}
          placeholder="購入時期、着用回数、保管状態などを入力してください。"
          rows={6}
          helperText="購入者が商品の状態を判断しやすい内容を入力してください。"
          counterText={`${description.length}/${DESCRIPTION_MAX_LENGTH}`}
          maxLength={DESCRIPTION_MAX_LENGTH}
          disabled={disabled}
          onChange={(event) => onDescriptionChange(event.currentTarget.value)}
        />
      </div>
    </section>
  );
}
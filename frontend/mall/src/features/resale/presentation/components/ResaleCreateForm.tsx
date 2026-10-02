// frontend/mall/src/features/resale/presentation/components/ResaleCreateForm.tsx

import Dropdown from "../../../../components/ui/Dropdown";
import Input from "../../../../components/ui/Input";
import Textbox from "../../../../components/ui/Textbox";

import {
  RESALE_CONDITION_OPTIONS,
  type ResaleCondition,
} from "../../../shared/types/resale";

export const RESALE_DESCRIPTION_MAX_LENGTH = 1000;

export type ResaleCreateFormProps = {
  formattedPrice: string;
  condition: ResaleCondition;
  description: string;
  disabled?: boolean;
  mobileComposerEnabled?: boolean;
  onPriceFocus?: () => void;
  onDescriptionFocus?: () => void;
  onPriceChange: (value: string) => void;
  onConditionChange: (value: ResaleCondition) => void;
  onDescriptionChange: (value: string) => void;
};

export default function ResaleCreateForm({
  formattedPrice,
  condition,
  description,
  disabled = false,
  mobileComposerEnabled = false,
  onPriceFocus,
  onDescriptionFocus,
  onPriceChange,
  onConditionChange,
  onDescriptionChange,
}: ResaleCreateFormProps) {
  return (
    <section className="resale-create-form">
      <div className="page-form">
        <Input
          label="販売価格"
          type="text"
          inputMode="numeric"
          value={formattedPrice}
          placeholder="例：12,000"
          helperText="半角数字で入力してください。"
          required
          readOnly={mobileComposerEnabled}
          disabled={disabled}
          onFocus={
            mobileComposerEnabled
              ? onPriceFocus
              : undefined
          }
          onChange={(event) => onPriceChange(event.currentTarget.value)}
        />

        <div className="page-form__field">
          <span className="page-form__label">商品の状態</span>

          <Dropdown
            buttonLabel={condition}
            items={RESALE_CONDITION_OPTIONS}
            selectedValue={condition}
            onSelect={onConditionChange}
            disabled={disabled}
          />
        </div>

        <Textbox
          label="説明文"
          value={description}
          placeholder="購入時期、着用回数、保管状態などを入力してください。"
          rows={6}
          helperText="購入者が商品の状態を判断しやすい内容を入力してください。"
          counterText={`${description.length}/${RESALE_DESCRIPTION_MAX_LENGTH}`}
          maxLength={RESALE_DESCRIPTION_MAX_LENGTH}
          readOnly={mobileComposerEnabled}
          disabled={disabled}
          onFocus={
            mobileComposerEnabled
              ? onDescriptionFocus
              : undefined
          }
          onChange={(event) => onDescriptionChange(event.currentTarget.value)}
        />
      </div>
    </section>
  );
}
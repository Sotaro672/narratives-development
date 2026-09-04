//frontend\amol\src\features\payment-method\components\PaymentMethodCardholderCard.tsx
type PaymentMethodCardholderCardProps = {
  cardholderName: string;
  isCreatingIntent: boolean;
  isLoading: boolean;
  onChangeCardholderName: (value: string) => void;
};

export default function PaymentMethodCardholderCard(
  props: PaymentMethodCardholderCardProps,
) {
  const {
    cardholderName,
    isCreatingIntent,
    isLoading,
    onChangeCardholderName,
  } = props;

  return (
    <div className="payment-method-page-card">
      <label
        className="payment-method-page-card__text"
        htmlFor="cardholderName"
      >
        <strong>カード名義人</strong>
      </label>

      <input
        id="cardholderName"
        type="text"
        value={cardholderName}
        onChange={(event) => onChangeCardholderName(event.target.value)}
        placeholder="例: TARO YAMADA"
        className="payment-method-page-input"
        autoComplete="cc-name"
        disabled={isCreatingIntent || isLoading}
      />

      <p className="payment-method-page-form__note">
        カードに記載されている名義人を入力してください。
      </p>
    </div>
  );
}
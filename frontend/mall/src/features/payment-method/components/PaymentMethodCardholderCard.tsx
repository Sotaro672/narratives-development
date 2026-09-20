// frontend/mall/src/features/payment-method/components/PaymentMethodCardholderCard.tsx

import Card from "../../../components/ui/Card";
import Input from "../../../components/ui/Input";

type PaymentMethodCardholderCardProps = {
  cardholderName: string;
  isCreatingIntent: boolean;
  isLoading: boolean;
  onChangeCardholderName: (value: string) => void;
};

export default function PaymentMethodCardholderCard({
  cardholderName,
  isCreatingIntent,
  isLoading,
  onChangeCardholderName,
}: PaymentMethodCardholderCardProps) {
  return (
    <Card variant="panel">
      <Input
        id="cardholderName"
        name="cardholderName"
        type="text"
        label="カード名義人"
        value={cardholderName}
        placeholder="例: TARO YAMADA"
        helperText="カードに記載されている名義人を入力してください。"
        autoComplete="cc-name"
        fullWidth
        disabled={isCreatingIntent || isLoading}
        onChange={(event) => onChangeCardholderName(event.target.value)}
      />
    </Card>
  );
}
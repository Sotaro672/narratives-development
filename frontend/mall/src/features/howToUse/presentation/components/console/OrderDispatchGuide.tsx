// frontend/mall/src/features/howToUse/presentation/components/console/OrderDispatchGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function OrderDispatchGuide() {
  return (
    <>
      <HowToUseSection title="商品の発送">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「注文」を開いてください。</HowToUseStep>
          <HowToUseStep>発送する注文を選択してください。</HowToUseStep>
          <HowToUseStep>注文内容と配送先住所を確認してください。</HowToUseStep>
          <HowToUseStep>画面右上にある発送ボタンを押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/order/dispatch-order.mp4" label="注文内容を確認し、商品の発送手続きを完了する手順" />

        <HowToUseNote title="発送通知メール">発送ボタンを押すと、購入客へ発送通知メールが送信されます。</HowToUseNote>
      </HowToUseSection>
    </>
  );
}
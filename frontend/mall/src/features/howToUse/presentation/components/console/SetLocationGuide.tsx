// frontend/mall/src/features/howToUse/presentation/components/console/SetLocationGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function SetLocationGuide() {
  return (
    <>
      <HowToUseSection title="在庫の保管場所を設定します。">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「配送」「保管場所」「保管場所を追加」を開いてください。</HowToUseStep>
          <HowToUseStep>ミントした在庫商品の保管場所の住所を入力し、保存ボタンを押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/inventory/set-inventory-location.mp4" label="在庫画面から在庫保管場所の登録画面を開く手順" />
      </HowToUseSection>
    </>
  );
}
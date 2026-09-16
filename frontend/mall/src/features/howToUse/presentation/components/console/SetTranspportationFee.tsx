// frontend/mall/src/features/howToUse/presentation/components/console/SetTranspportationFee.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function SetTranspportationFee() {
  return (
    <>
      <HowToUseSection title="配送料金体系の設定">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「配送」「配送料金」「配送料金を作成」を開いてください。</HowToUseStep>
          <HowToUseStep>料金設定画面に地域・都道府県・島嶼部への配送料金を入力してください。</HowToUseStep>
          <HowToUseStep>料金体系に名前を付け、保存ボタンを押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/inventory/set-transportation-fee.mp4" label="配送画面から配送料金体系を設定して保存する手順" />
      </HowToUseSection>
    </>
  );
}
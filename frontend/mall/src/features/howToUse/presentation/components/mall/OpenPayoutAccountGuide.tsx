// frontend/mall/src/features/howToUse/presentation/components/mall/OpenPayoutAccountGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function OpenPayoutAccountGuide() {
  return (
    <HowToUseSection title="売上受取口座">
      <div className="how-to-use-mobile-guide">
        <HowToUseVideo
          storagePath="mall/payout/open-payout-account.mp4"
          label="フリマの売上を受け取るための口座を登録する手順"
          variant="iphone-12-pro"
        />

        <HowToUseStepList>
          <HowToUseStep>ウォレット画面から売上受取口座の設定画面を開いてください。</HowToUseStep>
          <HowToUseStep>売上を受け取る銀行口座の情報を入力してください。</HowToUseStep>
          <HowToUseStep>入力内容を確認し、口座を登録してください。</HowToUseStep>
        </HowToUseStepList>
      </div>

      <HowToUseNote title="フリマの売上受取">
        フリマで商品が売れた際の売上金を受け取るため、事前に売上受取口座を登録してください。
      </HowToUseNote>
    </HowToUseSection>
  );
}
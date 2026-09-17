// frontend/mall/src/features/howToUse/presentation/components/mall/TradeGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function TradeGuide() {
  return (
    <>
      <HowToUseSection title="商品を発送する">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/trade/dispatch-trade.mp4"
            label="フリマ取引で配送会社と箱サイズを選択し、商品を発送する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>成立した取引を開き、発送画面へ進んでください。</HowToUseStep>
            <HowToUseStep>商品の発送に使用する配送会社を選択してください。</HowToUseStep>
            <HowToUseStep>梱包後の箱の3辺合計に合わせて箱サイズを選択してください。</HowToUseStep>
            <HowToUseStep>選択した配送会社、箱サイズ、配送料を確認してください。</HowToUseStep>
            <HowToUseStep>内容に問題がなければ「発送を確定」を押してください。</HowToUseStep>
            <HowToUseStep>発送処理が完了すると取引のチャット画面へ戻ります。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="配送料について">
          配送料は配送先にかかわらず全国一律で、選択した箱サイズによって決まります。日本郵便とヤマト運輸のどちらを選択しても、同じ箱サイズであれば配送料は同額です。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}
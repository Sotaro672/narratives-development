// frontend/mall/src/features/howToUse/presentation/components/mall/ReviewResaleGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ReviewResaleGuide() {
  return (
    <>
      <HowToUseSection title="マーケットを見る">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/market/view-market.mp4"
            label="Mallのマーケット一覧から出品商品を選択し、商品詳細を確認する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>Mallのマーケットを開いてください。</HowToUseStep>
            <HowToUseStep>マーケット一覧から確認する商品を選択してください。</HowToUseStep>
            <HowToUseStep>商品状態の写真、販売価格、商品の状態、説明文などの出品内容を確認してください。</HowToUseStep>
            <HowToUseStep>購入する場合は「カートに入れる」を押してください。以降の手順は通常の商品購入と同じです。</HowToUseStep>
            <HowToUseStep>出品詳細画面から出品者の他の出品商品を見ることもできます。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>

      <HowToUseSection title="出品者とチャットする">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/market/post-review-on-resale.mp4"
            label="Mallのマーケット出品詳細から値下げコメントを投稿する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>マーケット出品画面にあるメッセージアイコンを押してください。</HowToUseStep>
            <HowToUseStep>出品者へ伝えるメッセージを投稿してください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}
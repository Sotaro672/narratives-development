// frontend/mall/src/features/howToUse/presentation/components/mall/PostProductReviewGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function PostProductReviewGuide() {
  return (
    <HowToUseSection title="レビュー投稿">
      <div className="how-to-use-mobile-guide">
        <HowToUseVideo
          storagePath="mall/review/post-product-review.mp4"
          label="購入した商品のレビューを投稿する手順"
          variant="iphone-12-pro"
        />

        <HowToUseStepList>
          <HowToUseStep>ウォレットからレビューを投稿する商品を開いてください。</HowToUseStep>
          <HowToUseStep>レビューを開き、商品の評価を選択してください。</HowToUseStep>
          <HowToUseStep>商品の感想や体験を入力してください。</HowToUseStep>
          <HowToUseStep>「投稿」を押すと、商品レビューが投稿されます。</HowToUseStep>
        </HowToUseStepList>
      </div>
    </HowToUseSection>
  );
}
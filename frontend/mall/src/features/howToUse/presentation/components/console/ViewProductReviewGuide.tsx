// frontend/mall/src/features/howToUse/presentation/components/console/ViewProductReviewGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ViewProductReviewGuide() {
  return (
    <HowToUseSection title="レビュー確認">
      <HowToUseStepList>
        <HowToUseStep>サイドバーから「レビュー」「商品」を押し、確認する商品行を押してください。</HowToUseStep>
        <HowToUseStep>商品に投稿されたレビューを確認できます。</HowToUseStep>
      </HowToUseStepList>

      <HowToUseVideo
        storagePath="console/review/view-product-review.mp4"
        label="Consoleから商品に投稿されたレビューを確認する手順"
      />
    </HowToUseSection>
  );
}
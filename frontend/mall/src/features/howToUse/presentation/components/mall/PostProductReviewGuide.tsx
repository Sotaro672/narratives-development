// frontend/mall/src/features/howToUse/presentation/components/mall/PostProductReviewGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
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
          <HowToUseStep>トークンコンテンツ画面の商品名タブを押してスキャン画面へ移動してください。またはスキャン画面でQRコードをスキャンしてください。</HowToUseStep>
          <HowToUseStep>スキャン画面にて商品に対する感想を投稿してください。</HowToUseStep>
        </HowToUseStepList>
      </div>

      <HowToUseNote title="購入者のみレビュー投稿が可能">
        商品レビューは、その商品を購入したユーザーのみ投稿できます。
      </HowToUseNote>
    </HowToUseSection>
  );
}
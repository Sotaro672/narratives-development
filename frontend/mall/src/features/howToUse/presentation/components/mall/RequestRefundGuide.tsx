// frontend/mall/src/features/howToUse/presentation/components/mall/RequestRefundGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function RequestRefundGuide() {
  return (
    <>
      <HowToUseSection title="返品申請">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/refund/request-refund.mp4" label="Mallの注文詳細から返品を申請する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>注文一覧から返品する注文を開いてください。</HowToUseStep>
            <HowToUseStep>返品する商品を確認し、返品申請を開いてください。</HowToUseStep>
            <HowToUseStep>商品の開封状態と返品理由を入力してください。</HowToUseStep>
            <HowToUseStep>入力内容を確認し、返品申請を送信してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="未開封商品の返品について">未開封として返品申請した後に商品のQRコードをスキャンすると、開封後での返品として自動的に更新されます。ご注意ください。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="返信入力">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/refund/refund-from-mall.mp4" label="ブランドから届いた返品申請への返信を確認し、Mallから返信する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>ブランドから返品申請の返答が届くと、ヘッダーのメッセージアイコンに未読カウンターが表示されます。</HowToUseStep>
            <HowToUseStep>ヘッダーのメッセージアイコンを押すと、返信を確認できます。</HowToUseStep>
            <HowToUseStep>フッターの返信アイコンを押して返信を入力してください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>

      <HowToUseSection title="お問い合わせの終了">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/refund/close-inquiry.mp4" label="返品申請に関するやり取りを確認し、お問い合わせを終了する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>ブランドから返金処理が行われるとメッセージ通知が届きます。</HowToUseStep>
            <HowToUseStep>返金内容を確認し、クローズするボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}
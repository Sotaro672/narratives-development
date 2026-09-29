// frontend/mall/src/features/howToUse/presentation/components/mall/PurchaseGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function PurchaseGuide() {
  return (
    <>
      <HowToUseSection id="purchase-order" title="商品の購入">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/purchase/purchase.mp4" label="Mallで商品を選択し、購入手続きを行う手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>購入する商品を開いてください。</HowToUseStep>
            <HowToUseStep>型番を選択し、カートに入れるボタンを押してください。</HowToUseStep>
            <HowToUseStep>配送先住所と支払方法を選択して購入ボタンを押してください。</HowToUseStep>
            <HowToUseStep>注文内容と支払金額を確認して支払ボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="注文確定メール">注文確定後、登録しているメールアドレス宛に注文確定メールが送信されます。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection id="purchase-token-transfer" title="トークンの受け取り">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/purchase/token-transfer.mp4" label="購入した商品のトークンをウォレットへ受け取る手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>商品到着後、商品のQRコードをスキャンしてください。</HowToUseStep>
            <HowToUseStep>スキャンした商品が購入商品と一致する場合、トークン移譲をしてもよいか確認されます。</HowToUseStep>
            <HowToUseStep>承認ボタンを押すとトークン移譲が開始されます。</HowToUseStep>
            <HowToUseStep>トークン移譲後は商品、トークンへのコメント投稿、トークンコンテンツの閲覧ができるようになります。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="トークン移譲後は返品不可">トークン移譲確認でも記載されている通り、トークン移譲後は返品申請ができません。商品を問題なく受け取ったという意思表示としてトークンを受け取ってください。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection id="purchase-resale-receipt" title="フリマ取引での受取">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/purchase/post-review-on-avatar.mp4" label="フリマで購入した商品のトークンを受け取り、取引相手のアバターへレビューを投稿する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>フリマで購入した商品が届いたら、商品のQRコードをスキャンしてください。</HowToUseStep>
            <HowToUseStep>スキャンした商品が購入商品と一致していることを確認し、トークン移譲を承認してください。</HowToUseStep>
            <HowToUseStep>トークンの受け取りが完了すると、取引相手のアバターを評価する画面へ移動します。</HowToUseStep>
            <HowToUseStep>取引について「良かった」または「残念だった」を選択してください。</HowToUseStep>
            <HowToUseStep>取引についてのコメントを入力し、「評価を投稿」を押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="トークンを受け取る前に商品を確認">フリマ取引では、商品の状態に問題がないことを確認してからトークンを受け取ってください。トークンの受け取り後は取引相手への評価へ進みます。</HowToUseNote>
      </HowToUseSection>
    </>
  );
}
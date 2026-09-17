// frontend/mall/src/features/howToUse/presentation/components/mall/PurchaseGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function PurchaseGuide() {
  return (
    <>
      <HowToUseSection title="商品の購入">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/purchase/purchase.mp4"
            label="Mallで商品を選択し、購入手続きを行う手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>購入する商品を開いてください。</HowToUseStep>
            <HowToUseStep>型番を選択し、カートに入れるボタンを押してください。</HowToUseStep>
            <HowToUseStep>配送先住所と支払方法を選択して購入ボタンを押してください。</HowToUseStep>
            <HowToUseStep>注文内容と支払金額を確認して支払ボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="注文確定メール">注文確定後、登録しているメールアドレス宛に注文確定メールが送信されます。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="トークンの受け取り">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/purchase/token-transfer.mp4"
            label="購入した商品のトークンをウォレットへ受け取る手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>商品到着後、商品のQRコードをスキャンしてください。</HowToUseStep>
            <HowToUseStep>スキャンした商品が購入商品と一致する場合、トークンがアバターへ渡されます。</HowToUseStep>
            <HowToUseStep>コンテンツを見るボタンを押すとトークンのメタデータコンテンツを閲覧することができます。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="所有者のみコンテンツ閲覧が可能">
          トークンコンテンツはトークン所有者のみ閲覧できるようにしています。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}
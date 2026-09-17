// frontend/mall/src/features/howToUse/presentation/components/mall/ListMarketGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ListMarketGuide() {
  return (
    <>
      <HowToUseSection title="フリマ出品画面を開く">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/resale/go-to-resale-create-page.mp4"
            label="Mallで所有しているトークンを選択し、フリマ出品画面を開く手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>マイページから「トークン」を開いてください。</HowToUseStep>
            <HowToUseStep>フリマへ出品するトークンを選択してください。</HowToUseStep>
            <HowToUseStep>トークン詳細画面の「出品」を押してください。</HowToUseStep>
            <HowToUseStep>フリマ出品画面が表示されたことを確認してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="売上受取口座">
          フリマへ出品するには売上受取口座の登録が必要です。未登録の場合は、出品ボタンを押した後に売上受取口座の登録画面へ移動します。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="フリマへ出品する">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/resale/list-on-market.mp4"
            label="Mallのフリマ出品画面で商品状態の写真、販売価格、商品の状態、説明文を入力して出品する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>商品の現在の状態が分かる写真を追加してください。</HowToUseStep>
            <HowToUseStep>販売価格を入力してください。</HowToUseStep>
            <HowToUseStep>商品の状態を選択してください。</HowToUseStep>
            <HowToUseStep>購入時期、使用回数、保管状態など必要に応じて説明文を入力してください。</HowToUseStep>
            <HowToUseStep>入力内容を確認し、「出品」を押してください。</HowToUseStep>
            <HowToUseStep>画像の転送と出品処理が完了するまで、そのままお待ちください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}
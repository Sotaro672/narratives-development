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
            <HowToUseStep>出品が完了するとウォレットページの出品パネルに出品内容が表示されます。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>

      <HowToUseSection title="出品内容を編集する">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/resale/edit-resale.mp4"
            label="Mallの出品詳細画面から販売価格、商品の状態、公開状態、商品状態の写真、説明文を編集する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページから出品パネルを開いてください。</HowToUseStep>
            <HowToUseStep>編集する出品を選択してください。</HowToUseStep>
            <HowToUseStep>出品詳細画面の「編集する」を押してください。</HowToUseStep>
            <HowToUseStep>必要に応じて商品状態の写真を追加または削除してください。</HowToUseStep>
            <HowToUseStep>販売価格、商品の状態、公開状態、説明文を編集してください。動画では公開停止へ変更しています。</HowToUseStep>
            <HowToUseStep>編集内容を確認し、「保存する」を押してください。</HowToUseStep>
            <HowToUseStep>更新が完了すると、編集した内容が出品詳細画面に反映されます。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="売却済みの商品">
          売却済みになった出品は編集できません。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="フリマを見る">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/resale/view-market.mp4"
            label="Mallのフリマ一覧から出品商品を選択し、商品詳細を確認する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>Mallのマーケットを開いてください。</HowToUseStep>
            <HowToUseStep>フリマ一覧から確認する商品を選択してください。</HowToUseStep>
            <HowToUseStep>商品状態の写真、販売価格、商品の状態、説明文などの出品内容を確認してください。</HowToUseStep>
            <HowToUseStep>購入する場合は「カートに入れる」を押してください。以降は購入と同じです。</HowToUseStep>
            <HowToUseStep>マーケット出品画面から出品者の他の出品商品を見ることができます。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}
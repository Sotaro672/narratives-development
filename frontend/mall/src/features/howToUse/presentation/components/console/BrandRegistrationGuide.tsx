// frontend/mall/src/features/howToUse/presentation/components/console/BrandRegistrationGuide.tsx

import brandRegistrationOpenImage from "../../../assets/console/brand-registration/brand-registration-open.png";
import brandRegistrationImagesImage from "../../../assets/console/brand-registration/brand-registration-images.png";
import brandRegistrationDetailsImage from "../../../assets/console/brand-registration/brand-registration-details.png";
import brandRegistrationSaveImage from "../../../assets/console/brand-registration/brand-registration-save.png";

import HowToUseFigure from "../common/HowToUseFigure";
import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";

export default function BrandRegistrationGuide() {
  return (
    <>
      <HowToUseSection title="ブランド登録画面の開け方">
        <HowToUseStepList>
          <HowToUseStep>
            サイドバーの「組織」をクリックしてください。
          </HowToUseStep>
          <HowToUseStep>
            「組織」をクリックした際に表示される「ブランド」をクリックしてください。
          </HowToUseStep>
          <HowToUseStep>
            ブランド管理画面右上にある「ブランド追加」をクリックしてください。
          </HowToUseStep>
        </HowToUseStepList>

        <HowToUseFigure
          src={brandRegistrationOpenImage}
          alt="サイドバーから組織、ブランドを選択し、ブランド追加ボタンをクリックする手順"
          caption="ブランド登録画面の開き方"
        />
      </HowToUseSection>

      <HowToUseSection title="ブランド登録手順">
        <HowToUseStepList>
          <HowToUseStep>
            ブランド担当者を選択してください。
          </HowToUseStep>
          <HowToUseStep>
            ブランド売上振込口座を選択してください。開発環境ではテスト口座で固定されています。
          </HowToUseStep>
          <HowToUseStep>
            ブランドアイコン画像をアップロードしてください。ブランドアイコンは出品画面やレビューへの返答時にブランド名と共に表示されます。
          </HowToUseStep>
          <HowToUseStep>
            ブランド背景画像をアップロードしてください。Mallのブランド商品一覧画面のトップに表示される画像です。ブランドの世界観や、お客様へ伝えたいメッセージを表現する画像を設定してください。
          </HowToUseStep>
        </HowToUseStepList>

        <HowToUseFigure
          src={brandRegistrationImagesImage}
          alt="ブランド担当者、売上振込口座、ブランドアイコン画像、ブランド背景画像を設定する画面"
          caption="担当者・口座・ブランド画像の設定"
        />

        <HowToUseStepList>
          <HowToUseStep>
            ブランド名、ブランド説明、会社サイトURL、ブランド責任者を入力・選択してください。
          </HowToUseStep>
        </HowToUseStepList>

        <HowToUseFigure
          src={brandRegistrationDetailsImage}
          alt="ブランド名、ブランド説明、会社サイトURL、ブランド責任者を入力する画面"
          caption="ブランド基本情報の入力"
        />

        <HowToUseStepList>
          <HowToUseStep>
            すべての入力が完了したら、画面右上にある「保存」ボタンをクリックしてください。
          </HowToUseStep>
        </HowToUseStepList>

        <HowToUseFigure
          src={brandRegistrationSaveImage}
          alt="ブランド登録画面右上の保存ボタンをクリックする手順"
          caption="入力完了後に保存"
        />

        <HowToUseNote title="ブランド専用ウォレット">
          ブランド登録と同時に、ブランド専用のブロックチェーンウォレットが開設されます。商品やトークン設計を登録する際にはブランド名義が必要です。
        </HowToUseNote>

        <HowToUseNote title="ガスについて">
          ミントや購入されたお客様へトークンを移譲する際に必要なガスは、必要量が自動で補給されます。ガス代は電子名札発行手数料として商品1点につき10円で請求されます。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}
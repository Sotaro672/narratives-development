// frontend/mall/src/features/howToUse/presentation/components/mall/PostCommentGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function PostCommentGuide() {
  return (
    <HowToUseSection title="コメント投稿">
      <div className="how-to-use-mobile-guide">
        <HowToUseVideo
          storagePath="mall/comment/post-comment-on-tokenBlueprint.mp4"
          label="所有しているトークンのコンテンツにコメントを投稿する手順"
          variant="iphone-12-pro"
        />

        <HowToUseStepList>
          <HowToUseStep>ウォレットからコメントを投稿するトークンを開いてください。</HowToUseStep>
          <HowToUseStep>画面下部のコメント入力欄にコメントを入力してください。</HowToUseStep>
          <HowToUseStep>「投稿」を押すと、トークンのコメント欄にコメントが投稿されます。</HowToUseStep>
        </HowToUseStepList>
      </div>
    </HowToUseSection>
  );
}
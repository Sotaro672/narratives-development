// frontend/admin/shell/src/pages/AvatarDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useAvatars } from "../features/avatar/presentation/hooks/useAvatars";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

import "./AvatarDetailPage.css";

export default function AvatarDetailPage() {
  const navigate = useNavigate();
  const { avatarId = "" } = useParams<{ avatarId?: string }>();
  const { avatars, loading, error, reload } = useAvatars();

  const avatar = avatars.find((item) => item.id === avatarId) ?? null;

  const renderAside = () => {
    if (loading) {
      return <p>アバター詳細を取得しています...</p>;
    }

    if (error) {
      return (
        <div role="alert">
          <p>アバター詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!avatar) {
      return <p role="alert">アバターが見つかりませんでした。</p>;
    }

    return (
      <section className="ui-detail-section avatar-detail-page">
        <dl className="ui-detail-definition-list">
          <div>
            <dt>ユーザー氏名</dt>
            <dd>{avatar.userName || "-"}</dd>
          </div>
          <div>
            <dt>プロフィール</dt>
            <dd>{avatar.profile || "-"}</dd>
          </div>
          <div>
            <dt>外部リンク</dt>
            <dd>{avatar.externalLink || "-"}</dd>
          </div>
          <div>
            <dt>通報数</dt>
            <dd>{avatar.reportCount}</dd>
          </div>
          <div>
            <dt>登録日時</dt>
            <dd>{formatDateTime(avatar.createdAt)}</dd>
          </div>
          <div>
            <dt>最終更新日時</dt>
            <dd>{formatDateTime(avatar.updatedAt)}</dd>
          </div>
        </dl>
      </section>
    );
  };

  return (
    <Page>
      <PageHeader
        title={avatar?.avatarName || "アバター詳細"}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/avatars")}
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
      />

      <DetailPageBody main={null} aside={renderAside()} />
    </Page>
  );
}
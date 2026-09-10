// frontend/admin/shell/src/pages/AvatarPage.tsx

import { useNavigate } from "react-router-dom";

import AvatarTable from "../features/avatar/presentation/components/AvatarTable";
import { useAvatars } from "../features/avatar/presentation/hooks/useAvatars";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";

export default function AvatarPage() {
  const navigate = useNavigate();
  const { avatars, loading, error, reload } = useAvatars();

  return (
    <Page>
      <PageHeader
        title="アバター"
        actions={
          <RefreshButton
            onClick={reload}
            loading={loading}
            title="リフレッシュ"
            ariaLabel="アバター一覧をリフレッシュ"
          />
        }
      />

      {loading && <p>アバター一覧を読み込んでいます。</p>}

      {!loading && error && (
        <p role="alert">
          アバター一覧の取得に失敗しました。{error}
        </p>
      )}

      {!loading && !error && (
        <AvatarTable
          avatars={avatars}
          onAvatarClick={(avatar) =>
            navigate(`/avatars/${encodeURIComponent(avatar.id)}`)
          }
        />
      )}
    </Page>
  );
}
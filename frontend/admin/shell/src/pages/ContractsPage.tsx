// frontend/admin/shell/src/pages/ContractsPage.tsx

import CompanyTable from "../features/company/presentation/components/CompanyTable";
import { useCompanies } from "../features/company/presentation/hooks/useCompanies";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";

export default function ContractsPage() {
  const { companies, loading, error, reload } = useCompanies();

  return (
    <Page>
      <PageHeader
        title="契約"
        actions={
          <RefreshButton
            onClick={reload}
            loading={loading}
            title="リフレッシュ"
            ariaLabel="企業一覧をリフレッシュ"
          />
        }
      />

      {loading && <p>企業一覧を読み込んでいます。</p>}

      {!loading && error && (
        <p role="alert">企業一覧の取得に失敗しました。{error}</p>
      )}

      {!loading && !error && <CompanyTable companies={companies} />}
    </Page>
  );
}
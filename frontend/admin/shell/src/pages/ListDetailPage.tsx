// frontend/admin/shell/src/pages/ListDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import ListDetailAside from "../features/list/presentation/components/ListDetailAside";
import ListDetailPriceList from "../features/list/presentation/components/ListDetailPriceList";
import ListDetailSummary from "../features/list/presentation/components/ListDetailSummary";
import { useListDetail } from "../features/list/presentation/hooks/useListDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import Tab, { type TabTone } from "../shared/ui/Tab/Tab";

import "./ListDetailPage.css";

function formatListStatus(status: string): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "停止中";
    default:
      return status || "-";
  }
}

function getListStatusTone(status: string): TabTone {
  switch (status) {
    case "listing":
      return "success";
    case "suspended":
      return "danger";
    default:
      return "neutral";
  }
}

export default function ListDetailPage() {
  const navigate = useNavigate();
  const { companyId = "", listId = "" } = useParams<{
    companyId?: string;
    listId?: string;
  }>();

  const { detail, loading, error, reload } = useListDetail(companyId, listId);

  const company = detail?.company ?? null;
  const list = detail?.list ?? null;

  const renderMain = () => {
    if (loading && !detail) {
      return <p>出品詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>出品詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!list) {
      return <p role="alert">出品情報を取得できませんでした。</p>;
    }

    return (
      <>
        <ListDetailSummary list={list} />
        <ListDetailPriceList prices={list.prices} />
      </>
    );
  };

  return (
    <Page>
      <PageHeader
        title={list?.title || list?.readableId || "出品詳細"}
        meta={
          list?.status ? (
            <Tab
              tone={getListStatusTone(list.status)}
              aria-label={`出品状態 ${formatListStatus(list.status)}`}
            >
              {formatListStatus(list.status)}
            </Tab>
          ) : undefined
        }
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate(-1)}
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

      {company && list ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <ListDetailAside
              company={company}
              list={list}
              onOpenProductBlueprint={() =>
                navigate(
                  `/contracts/${encodeURIComponent(companyId)}/product-blueprints/${encodeURIComponent(list.productBlueprintId)}`,
                )
              }
              onOpenTokenBlueprint={() =>
                navigate(
                  `/contracts/${encodeURIComponent(companyId)}/token-blueprints/${encodeURIComponent(list.tokenBlueprintId)}`,
                )
              }
              onOpenReport={() =>
                navigate(`/reports/${encodeURIComponent(`list_${list.id}`)}`)
              }
            />
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}
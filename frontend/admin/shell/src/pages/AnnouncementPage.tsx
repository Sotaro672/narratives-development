// frontend/admin/shell/src/pages/AnnouncementPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useContractDetail } from "../features/company/presentation/hooks/useContractDetail";
import Page, {
  DetailPageBody,
  PageHeader,
} from "../shared/ui/Page/Page";
import Tab from "../shared/ui/Tab/Tab";
import { formatDateTime } from "../shared/util/dateFormat";

export default function AnnouncementPage() {
  const navigate = useNavigate();

  const {
    companyId = "",
    announcementId = "",
  } = useParams<{
    companyId?: string;
    announcementId?: string;
  }>();

  const {
    detail,
    loading,
    error,
    reload,
  } = useContractDetail(companyId);

  const company = detail?.company ?? null;

  const announcement =
    detail?.announcements.find(
      (item) => item.id === announcementId,
    ) ?? null;

  const renderMain = () => {
    if (loading && !detail) {
      return <p>告知詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>告知詳細を取得できませんでした。</p>
          <p>{error}</p>

          <button
            type="button"
            onClick={() => void reload()}
          >
            再読み込み
          </button>
        </div>
      );
    }

    if (!announcement) {
      return (
        <p role="alert">
          告知情報を取得できませんでした。
        </p>
      );
    }

    return (
      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-detail-definition-list--rows">
          <div>
            <dt>対象トークン</dt>
            <dd>{announcement.tokenName || "-"}</dd>
          </div>

          <div>
            <dt>送信対象数</dt>
            <dd>
              {announcement.targetAvatarCount.toLocaleString()}
            </dd>
          </div>
        </dl>
      </section>
    );
  };

  return (
    <Page>
      <PageHeader
        title={announcement?.title || "告知詳細"}
        meta={
          announcement ? (
            <Tab
              tone={
                announcement.published
                  ? "success"
                  : "warning"
              }
              aria-label={`告知状態 ${
                announcement.published
                  ? "公開済み"
                  : "下書き"
              }`}
            >
              {announcement.published
                ? "公開済み"
                : "下書き"}
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

      {company && announcement ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
                <dt>企業名</dt>
                <dd>{company.name || "-"}</dd>

                <dt>告知ID</dt>
                <dd>{announcement.id || "-"}</dd>

                <dt>対象トークンID</dt>
                <dd>
                  {announcement.tokenBlueprintId || "-"}
                </dd>

                <dt>作成日時</dt>
                <dd>
                  {announcement.createdAt
                    ? formatDateTime(
                        announcement.createdAt,
                      )
                    : "-"}
                </dd>

                <dt>最終更新日時</dt>
                <dd>
                  {announcement.updatedAt
                    ? formatDateTime(
                        announcement.updatedAt,
                      )
                    : "-"}
                </dd>
              </dl>
            </section>
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}
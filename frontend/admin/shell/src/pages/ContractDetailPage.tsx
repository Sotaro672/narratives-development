// frontend/admin/shell/src/pages/ContractDetailPage.tsx

import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import ContractListTable from "../features/company/presentation/components/ContractListTable";
import ContractProductBlueprintTable from "../features/company/presentation/components/ContractProductBlueprintTable";
import ContractTokenBlueprintTable from "../features/company/presentation/components/ContractTokenBlueprintTable";
import { useContractDetail } from "../features/company/presentation/hooks/useContractDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

import "./ContractDetailPage.css";

type ContractDetailTab = "lists" | "tokenBlueprints" | "productBlueprints";

export default function ContractDetailPage() {
  const navigate = useNavigate();
  const { companyId = "" } = useParams<{ companyId?: string }>();
  const { detail, loading, error, reload } = useContractDetail(companyId);
  const [activeTab, setActiveTab] = useState<ContractDetailTab>("lists");

  const company = detail?.company ?? null;

  const renderMain = () => {
    if (loading && !detail) {
      return <p>契約詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>契約詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!detail) {
      return <p role="alert">企業情報を取得できませんでした。</p>;
    }

    return (
      <div className="contract-detail">
        <div
          className="contract-detail__tabs"
          role="tablist"
          aria-label="契約関連情報"
        >
          <button
            id="contract-detail-tab-lists"
            type="button"
            role="tab"
            className={[
              "contract-detail__tab",
              activeTab === "lists" ? "contract-detail__tab--active" : "",
            ].filter(Boolean).join(" ")}
            aria-selected={activeTab === "lists"}
            aria-controls="contract-detail-panel-lists"
            onClick={() => setActiveTab("lists")}
          >
            出品
          </button>

          <button
            id="contract-detail-tab-token-blueprints"
            type="button"
            role="tab"
            className={[
              "contract-detail__tab",
              activeTab === "tokenBlueprints"
                ? "contract-detail__tab--active"
                : "",
            ].filter(Boolean).join(" ")}
            aria-selected={activeTab === "tokenBlueprints"}
            aria-controls="contract-detail-panel-token-blueprints"
            onClick={() => setActiveTab("tokenBlueprints")}
          >
            トークン設計
          </button>

          <button
            id="contract-detail-tab-product-blueprints"
            type="button"
            role="tab"
            className={[
              "contract-detail__tab",
              activeTab === "productBlueprints"
                ? "contract-detail__tab--active"
                : "",
            ].filter(Boolean).join(" ")}
            aria-selected={activeTab === "productBlueprints"}
            aria-controls="contract-detail-panel-product-blueprints"
            onClick={() => setActiveTab("productBlueprints")}
          >
            商品設計
          </button>
        </div>

        {activeTab === "lists" && (
          <div
            id="contract-detail-panel-lists"
            className="contract-detail__panel"
            role="tabpanel"
            aria-labelledby="contract-detail-tab-lists"
          >
            <ContractListTable lists={detail.lists} />
          </div>
        )}

        {activeTab === "tokenBlueprints" && (
          <div
            id="contract-detail-panel-token-blueprints"
            className="contract-detail__panel"
            role="tabpanel"
            aria-labelledby="contract-detail-tab-token-blueprints"
          >
            <ContractTokenBlueprintTable
              tokenBlueprints={detail.tokenBlueprints}
            />
          </div>
        )}

        {activeTab === "productBlueprints" && (
          <div
            id="contract-detail-panel-product-blueprints"
            className="contract-detail__panel"
            role="tabpanel"
            aria-labelledby="contract-detail-tab-product-blueprints"
          >
            <ContractProductBlueprintTable
              productBlueprints={detail.productBlueprints}
            />
          </div>
        )}
      </div>
    );
  };

  return (
    <Page>
      <PageHeader
        title={company?.name || "契約詳細"}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/contracts")}
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

      {company ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <h2 className="ui-detail-section__title">企業情報</h2>

              <dl className="ui-detail-definition-list">
                <dt>代表者</dt>
                <dd>{company.representativeName || "-"}</dd>

                <dt>登録日時</dt>
                <dd>{formatDateTime(company.createdAt)}</dd>

                <dt>最終更新日</dt>
                <dd>{formatDateTime(company.updatedAt)}</dd>
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
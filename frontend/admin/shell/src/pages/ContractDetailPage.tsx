// frontend/admin/shell/src/pages/ContractDetailPage.tsx

import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import BrandTable from "../features/brand/presentation/components/BrandTable";
import ContractAnnouncementTable from "../features/company/presentation/components/ContractAnnouncementTable";
import ContractListTable from "../features/list/presentation/components/ContractListTable";
import ContractProductBlueprintTable from "../features/company/presentation/components/ContractProductBlueprintTable";
import ContractTokenBlueprintTable from "../features/company/presentation/components/ContractTokenBlueprintTable";
import { useContractDetail } from "../features/company/presentation/hooks/useContractDetail";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

import "./ContractDetailPage.css";

type ContractDetailTab =
  | "brands"
  | "announcements"
  | "lists"
  | "tokenBlueprints"
  | "productBlueprints";

export default function ContractDetailPage() {
  const navigate = useNavigate();
  const { companyId = "" } = useParams<{ companyId?: string }>();
  const { detail, loading, error, reload } = useContractDetail(companyId);
  const [activeTab, setActiveTab] = useState<ContractDetailTab>("brands");

  const company = detail?.company ?? null;

  const handleBrandClick = (brandId: string) => {
    if (!companyId || !brandId) return;
    navigate(`/contracts/${encodeURIComponent(companyId)}/brands/${encodeURIComponent(brandId)}`);
  };

  const handleListClick = (listId: string) => {
    if (!companyId || !listId) return;
    navigate(`/contracts/${encodeURIComponent(companyId)}/lists/${encodeURIComponent(listId)}`);
  };

  const handleTokenBlueprintClick = (tokenBlueprintId: string) => {
    if (!companyId || !tokenBlueprintId) return;
    navigate(`/contracts/${encodeURIComponent(companyId)}/token-blueprints/${encodeURIComponent(tokenBlueprintId)}`);
  };

  const handleProductBlueprintClick = (productBlueprintId: string) => {
    if (!companyId || !productBlueprintId) return;
    navigate(`/contracts/${encodeURIComponent(companyId)}/product-blueprints/${encodeURIComponent(productBlueprintId)}`);
  };

  const renderMain = () => {
    if (loading && !detail) return <p>契約詳細を取得しています...</p>;

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

    if (!detail) return <p role="alert">企業情報を取得できませんでした。</p>;

    return (
      <div className="contract-detail">
        <div className="contract-detail__tabs" role="tablist" aria-label="契約関連情報">
          <button
            id="contract-detail-tab-brands"
            type="button"
            role="tab"
            className={[
              "contract-detail__tab",
              activeTab === "brands" ? "contract-detail__tab--active" : "",
            ].filter(Boolean).join(" ")}
            aria-selected={activeTab === "brands"}
            aria-controls="contract-detail-panel-brands"
            onClick={() => setActiveTab("brands")}
          >
            ブランド
          </button>

          <button
            id="contract-detail-tab-announcements"
            type="button"
            role="tab"
            className={[
              "contract-detail__tab",
              activeTab === "announcements" ? "contract-detail__tab--active" : "",
            ].filter(Boolean).join(" ")}
            aria-selected={activeTab === "announcements"}
            aria-controls="contract-detail-panel-announcements"
            onClick={() => setActiveTab("announcements")}
          >
            告知
          </button>

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
              activeTab === "tokenBlueprints" ? "contract-detail__tab--active" : "",
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
              activeTab === "productBlueprints" ? "contract-detail__tab--active" : "",
            ].filter(Boolean).join(" ")}
            aria-selected={activeTab === "productBlueprints"}
            aria-controls="contract-detail-panel-product-blueprints"
            onClick={() => setActiveTab("productBlueprints")}
          >
            商品設計
          </button>
        </div>

        {activeTab === "brands" && (
          <div
            id="contract-detail-panel-brands"
            className="contract-detail__panel"
            role="tabpanel"
            aria-labelledby="contract-detail-tab-brands"
          >
            <BrandTable
              brands={detail.brands}
              onBrandClick={(brand) => handleBrandClick(brand.id)}
            />
          </div>
        )}

        {activeTab === "announcements" && (
          <div
            id="contract-detail-panel-announcements"
            className="contract-detail__panel"
            role="tabpanel"
            aria-labelledby="contract-detail-tab-announcements"
          >
            <ContractAnnouncementTable announcements={detail.announcements} />
          </div>
        )}

        {activeTab === "lists" && (
          <div
            id="contract-detail-panel-lists"
            className="contract-detail__panel"
            role="tabpanel"
            aria-labelledby="contract-detail-tab-lists"
          >
            <ContractListTable
              lists={detail.lists}
              onListClick={(list) => handleListClick(list.id)}
            />
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
              onTokenBlueprintClick={(tokenBlueprint) =>
                handleTokenBlueprintClick(tokenBlueprint.id)
              }
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
              onProductBlueprintClick={(productBlueprint) =>
                handleProductBlueprintClick(productBlueprint.id)
              }
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
        <section className="contract-detail__company-info">
          <dl className="ui-detail-definition-list ui-detail-definition-list--rows contract-detail__company-details">
            <div>
              <dt>代表者</dt>
              <dd>{company.representativeName || "-"}</dd>
            </div>
            <div>
              <dt>登録日時</dt>
              <dd>{formatDateTime(company.createdAt)}</dd>
            </div>
            <div>
              <dt>最終更新日</dt>
              <dd>{formatDateTime(company.updatedAt)}</dd>
            </div>
          </dl>
        </section>
      ) : null}

      {renderMain()}
    </Page>
  );
}
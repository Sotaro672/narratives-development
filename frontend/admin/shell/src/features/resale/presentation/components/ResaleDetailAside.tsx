// frontend/admin/shell/src/features/resale/presentation/components/ResaleDetailAside.tsx

import { useNavigate } from "react-router-dom";

import type { Resale } from "../../../../shared/type/resale";
import TextLink from "../../../../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ResaleDetailAsideProps = {
  resale: Resale | null;
  loading: boolean;
  error: string | null;
  onReload: () => void | Promise<void>;
  returnReportCaseId?: string;
  returnReportLoading?: boolean;
  returnReportError?: string | null;
};

export default function ResaleDetailAside({
  resale,
  loading,
  error,
  onReload,
  returnReportCaseId,
  returnReportLoading,
  returnReportError,
}: ResaleDetailAsideProps) {
  const navigate = useNavigate();

  if (loading) {
    return <p>Resaleを取得しています...</p>;
  }

  if (error) {
    return (
      <div role="alert">
        <p>Resaleを取得できませんでした。</p>
        <p>{error}</p>
        <button type="button" onClick={() => void onReload()}>
          再読み込み
        </button>
      </div>
    );
  }

  if (!resale) {
    return <p role="alert">Resaleが見つかりませんでした。</p>;
  }

  const showReturnReport =
    returnReportLoading !== undefined ||
    returnReportError !== undefined ||
    returnReportCaseId !== undefined;

  return (
    <section className="ui-detail-section">
      <dl className="ui-detail-definition-list ui-detail-definition-list--rows">
        <div>
          <dt>商品名</dt>
          <dd>
            <TextLink
              tone="inherit"
              onClick={() =>
                navigate(`/contracts/${encodeURIComponent(resale.companyId)}/product-blueprints/${encodeURIComponent(resale.productBlueprintId)}`)
              }
            >
              {resale.productName || "-"}
            </TextLink>
          </dd>
        </div>

        <div>
          <dt>トークン名</dt>
          <dd>
            <TextLink
              tone="inherit"
              onClick={() =>
                navigate(`/contracts/${encodeURIComponent(resale.companyId)}/token-blueprints/${encodeURIComponent(resale.tokenBlueprintId)}`)
              }
            >
              {resale.tokenName || "-"}
            </TextLink>
          </dd>
        </div>

        <div>
          <dt>価格</dt>
          <dd>{resale.price.toLocaleString("ja-JP")}円</dd>
        </div>

        <div>
          <dt>商品の状態</dt>
          <dd>{resale.condition || "-"}</dd>
        </div>

        <div>
          <dt>通報数</dt>
          <dd>
            {resale.reportCount > 0 && resale.reportCaseId ? (
              <TextLink
                tone="accent"
                onClick={() =>
                  navigate(`/reports/${encodeURIComponent(resale.reportCaseId)}`)
                }
              >
                {resale.reportCount.toLocaleString("ja-JP")}
              </TextLink>
            ) : (
              resale.reportCount.toLocaleString("ja-JP")
            )}
          </dd>
        </div>

        {showReturnReport ? (
          <div>
            <dt>返品報告</dt>
            <dd>
              {returnReportLoading ? (
                "確認中..."
              ) : returnReportError ? (
                "-"
              ) : returnReportCaseId ? (
                <TextLink
                  tone="accent"
                  onClick={() =>
                    navigate(`/reports/${encodeURIComponent(returnReportCaseId)}`)
                  }
                >
                  あり
                </TextLink>
              ) : (
                "なし"
              )}
            </dd>
          </div>
        ) : null}

        <div>
          <dt>登録日時</dt>
          <dd>{formatDateTime(resale.createdAt)}</dd>
        </div>

        <div>
          <dt>最終更新日時</dt>
          <dd>{resale.updatedAt ? formatDateTime(resale.updatedAt) : "-"}</dd>
        </div>
      </dl>
    </section>
  );
}
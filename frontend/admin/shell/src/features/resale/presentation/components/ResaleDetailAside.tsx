// frontend/admin/shell/src/features/resale/presentation/components/ResaleDetailAside.tsx

import { useNavigate } from "react-router-dom";

import type { AvatarResale } from "../../../../shared/type/avatar";
import TextLink from "../../../../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../../../../shared/util/dateFormat";

type ResaleDetailAsideProps = {
  resale: AvatarResale | null;
  loading: boolean;
  error: string | null;
  onReload: () => void | Promise<void>;
};

export default function ResaleDetailAside({
  resale,
  loading,
  error,
  onReload,
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
        <button type="button" onClick={() => void onReload()}>再読み込み</button>
      </div>
    );
  }

  if (!resale) {
    return <p role="alert">Resaleが見つかりませんでした。</p>;
  }

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
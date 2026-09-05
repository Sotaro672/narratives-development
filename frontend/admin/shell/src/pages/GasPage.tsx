// frontend/admin/shell/src/pages/GasPage.tsx

import { useGasBalance } from "../features/gas/hooks/useGasBalance";
import Page from "../shared/ui/Page/Page";

export default function GasPage() {
  const { balance, loading, error, reload } = useGasBalance();

  return (
    <Page>
      <div>
        <h1>ガス</h1>
        <p>マスターウォレットのガス残高を確認します。</p>
      </div>

      <section>
        <div>
          <h2>マスターウォレット</h2>
          <button type="button" onClick={() => void reload()} disabled={loading}>
            {loading ? "更新中..." : "更新"}
          </button>
        </div>

        {loading && !balance ? <p>ガス残高を取得しています...</p> : null}

        {error ? (
          <div>
            <p>ガス残高を取得できませんでした。</p>
            <p>{error}</p>
          </div>
        ) : null}

        {balance ? (
          <dl>
            <div>
              <dt>残高</dt>
              <dd>{balance.balanceSol.toLocaleString(undefined, { maximumFractionDigits: 9 })} SOL</dd>
            </div>
            <div>
              <dt>Lamports</dt>
              <dd>{balance.balanceLamports}</dd>
            </div>
            <div>
              <dt>ネットワーク</dt>
              <dd>{balance.cluster}</dd>
            </div>
            <div>
              <dt>ウォレットアドレス</dt>
              <dd>{balance.address}</dd>
            </div>
          </dl>
        ) : null}
      </section>
    </Page>
  );
}
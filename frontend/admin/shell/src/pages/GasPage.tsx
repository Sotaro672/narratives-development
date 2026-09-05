// frontend/admin/shell/src/pages/GasPage.tsx

import { useGasBalance } from "../features/gas/hooks/useGasBalance";
import CopyButton from "../shared/ui/CopyButton/CopyButton";
import ExternalLinkButton from "../shared/ui/ExternalLinkButton/ExternalLinkButton";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";

export default function GasPage() {
  const { balance, loading, error, reload } = useGasBalance();

  return (
    <Page>
      <PageHeader
        title="ガス"
        actions={
          <RefreshButton
            onClick={reload}
            loading={loading}
            title="リフレッシュ"
            ariaLabel="リフレッシュ"
          />
        }
      />

      <section>
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
              <dd>
                {balance.balanceSol.toLocaleString(undefined, {
                  maximumFractionDigits: 9,
                })}{" "}
                SOL
              </dd>
            </div>

            <div>
              <dt>ネットワーク</dt>
              <dd>{balance.cluster}</dd>
            </div>

            <div>
              <dt>ウォレットアドレス</dt>
              <dd>
                {balance.address}
                <CopyButton value={balance.address} />
              </dd>
            </div>
          </dl>
        ) : null}
      </section>

      <section>
        <ExternalLinkButton
          href="https://faucet.solana.com/"
          title="Solana Faucetを開く"
          ariaLabel="Solana Faucetを開く"
        >
          Faucet
        </ExternalLinkButton>
      </section>
    </Page>
  );
}
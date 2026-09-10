// frontend/admin/shell/src/pages/GasPage.tsx

import { useGasBalance } from "../features/gas/hooks/useGasBalance";
import Button from "../shared/ui/Button/Button";
import CopyButton from "../shared/ui/CopyButton/CopyButton";
import ExternalLinkButton from "../shared/ui/ExternalLinkButton/ExternalLinkButton";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";

import "./GasPage.css";

export default function GasPage() {
  const { balance, loading, error, reload } = useGasBalance();

  return (
    <Page>
      <PageHeader
        title="ガス"
        actions={
          <>
            {balance ? (
              <Button variant="secondary" size="sm" title={`ネットワーク: ${balance.cluster}`} aria-label={`ネットワーク: ${balance.cluster}`}>
                {balance.cluster}
              </Button>
            ) : null}
            <RefreshButton onClick={reload} loading={loading} title="リフレッシュ" ariaLabel="リフレッシュ" />
          </>
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
          <dl className="gas-page__details">
            <div>
              <dt>残高</dt>
              <dd>{balance.balanceSol.toLocaleString(undefined, { maximumFractionDigits: 9 })} SOL</dd>
            </div>

            <div>
              <dt>ウォレットアドレス</dt>
              <dd>
                <span className="gas-page__wallet-address">{balance.address}</span>
                <CopyButton value={balance.address} />
                <ExternalLinkButton
                  href="https://faucet.solana.com/"
                  title="Solana Faucetを開く"
                  ariaLabel="Solana Faucetを開く"
                  className="gas-page__faucet-button"
                >
                  Faucet
                </ExternalLinkButton>
              </dd>
            </div>
          </dl>
        ) : null}
      </section>
    </Page>
  );
}
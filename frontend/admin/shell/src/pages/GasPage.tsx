// frontend/admin/shell/src/pages/GasPage.tsx

import { useCallback } from "react";
import { useNavigate } from "react-router-dom";

import { useGasBalance } from "../features/gas/hooks/useGasBalance";
import { useMints } from "../features/mint/hooks/useMints";
import MintTable from "../features/mint/presentation/components/MintTable";
import type { Mint } from "../shared/type/mint";
import Button from "../shared/ui/Button/Button";
import CopyButton from "../shared/ui/CopyButton/CopyButton";
import ExternalLinkButton from "../shared/ui/ExternalLinkButton/ExternalLinkButton";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";

import "./GasPage.css";

export default function GasPage() {
  const navigate = useNavigate();
  const { balance, loading: gasLoading, error: gasError, reload: reloadGas } = useGasBalance();
  const { mints, loading: mintsLoading, error: mintsError, reload: reloadMints } = useMints();

  const reload = useCallback(async () => {
    await Promise.all([reloadGas(), reloadMints()]);
  }, [reloadGas, reloadMints]);

  const handleMintClick = useCallback(
    (mint: Mint) => {
      navigate(`/gas/mints/${encodeURIComponent(mint.id)}`);
    },
    [navigate],
  );

  const loading = gasLoading || mintsLoading;

  return (
    <Page>
      <PageHeader
        title="ミント"
        meta={!mintsError ? `${mints.length}件` : undefined}
        actions={
          <>
            {balance ? (
              <Button
                variant="secondary"
                size="sm"
                title={`ネットワーク: ${balance.cluster}`}
                aria-label={`ネットワーク: ${balance.cluster}`}
              >
                {balance.cluster}
              </Button>
            ) : null}

            <RefreshButton
              onClick={reload}
              loading={loading}
              title="リフレッシュ"
              ariaLabel="ガスとMint一覧をリフレッシュ"
            />
          </>
        }
      />

      <section>
        {gasLoading && !balance ? <p>ガス残高を取得しています...</p> : null}

        {gasError ? (
          <div>
            <p>ガス残高を取得できませんでした。</p>
            <p>{gasError}</p>
          </div>
        ) : null}

        {balance ? (
          <dl className="ui-detail-definition-list ui-detail-definition-list--rows gas-page__details">
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

      <section className="gas-page__mints">
        {mintsLoading && mints.length === 0 ? <p>Mint一覧を取得しています...</p> : null}

        {!mintsLoading && mintsError ? (
          <p role="alert">Mint一覧を取得できませんでした。{mintsError}</p>
        ) : null}

        {!mintsError && (mints.length > 0 || !mintsLoading) ? (
          <MintTable mints={mints} onMintClick={handleMintClick} />
        ) : null}
      </section>
    </Page>
  );
}
// frontend/console/shell/src/pages/mintManagement.tsx

import List from "../layout/List/List";
import { useMintRequestManagement } from "../features/mint/presentation/hook/useMintRequestManagement";

import "../styles/mintRequest.css";

export default function MintRequestManagementPage() {
  const {
    headers,
    rows,
    onReset,
    isResetting,
    handleRowClick,
    handleRowKeyDown,
  } = useMintRequestManagement();

  return (
    <div className="p-0">
      <List
        title="ミント申請一覧"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        isResetting={isResetting}
        onReset={onReset}
      >
        {rows.map((row) => {
          /**
           * mints.createdByNameのみを使用する。
           */
          const requesterName = row.createdByName ?? "-";

          /**
           * Mint完了時だけ実行日時を表示する。
           */
          const mintedAtLabel =
            row.status === "minted"
              ? row.mintedAt ?? "-"
              : "-";

          /**
           * tokenNameを優先して表示し、
           * 存在しない場合はtokenBlueprintIdを表示する。
           */
          const tokenLabel =
            row.tokenName ??
            row.tokenBlueprintId ??
            "-";

          const productName =
            row.productName ?? "-";

          return (
            <tr
              key={row.productionId}
              onClick={() =>
                handleRowClick(
                  row.productionId,
                )
              }
              style={{
                cursor: "pointer",
              }}
              tabIndex={0}
              onKeyDown={(event) =>
                handleRowKeyDown(
                  event,
                  row.productionId,
                )
              }
              aria-label={`ミント申請 ${productName} の詳細へ`}
            >
              <td>
                <span className="truncate">
                  {tokenLabel}
                </span>
              </td>

              <td>
                <span className="truncate">
                  {productName}
                </span>
              </td>

              <td>
                {row.mintQuantity}
              </td>

              <td>
                {row.productionQuantity}
              </td>

              <td>
                {row.status === "minted" ? (
                  <span className="mint-badge is-done">
                    {row.statusLabel}
                  </span>
                ) : row.status === "minting" ? (
                  <span className="mint-badge is-minting">
                    {row.statusLabel}
                  </span>
                ) : row.status === "requested" ? (
                  <span className="mint-badge is-requested">
                    {row.statusLabel}
                  </span>
                ) : (
                  <span className="mint-badge is-planned">
                    {row.statusLabel}
                  </span>
                )}
              </td>

              <td>
                {requesterName}
              </td>

              <td>
                {mintedAtLabel}
              </td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}
// frontend/mintRequest/src/presentation/pages/mintRequestManagement.tsx
import List from "../../../../shell/src/layout/List/List";
import "../styles/mintRequest.css";
import { useMintRequestManagement } from "../hook/useMintRequestManagement";

export default function MintRequestManagementPage() {
  const { headers, rows, onReset, handleRowClick, handleRowKeyDown } =
    useMintRequestManagement();

  return (
    <div className="p-0">
      <List
        title="ミントリクエスト管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={onReset}
      >
        {rows.map((r) => {
          // ★ リクエスト者: mints.createdByName のみを使用（無ければ "-"）
          const requesterName = (r as any).createdByName ?? "-";

          // ★ ミント日時: minted 状態のときだけ mintedAt を表示（それ以外は "-"）
          const mintedAtLabel = r.status === "minted" ? r.mintedAt ?? "-" : "-";

          return (
            <tr
              key={r.id}
              onClick={() => handleRowClick(r.id)}
              style={{ cursor: "pointer" }}
              tabIndex={0}
              onKeyDown={(e) => handleRowKeyDown(e, r.id)}
              aria-label={`ミント申請 ${r.productName} の詳細へ`}
            >
              <td>
                <span className="lp-brand-pill">{r.tokenBlueprintId}</span>
              </td>
              <td>
                <span className="lp-brand-pill">{r.productName}</span>
              </td>
              <td>{r.mintQuantity}</td>
              <td>{r.productionQuantity ?? "-"}</td>
              <td>
                {r.status === "minted" ? (
                  <span className="mint-badge is-done">{r.statusLabel}</span>
                ) : r.status === "requested" ? (
                  <span className="mint-badge is-requested">{r.statusLabel}</span>
                ) : (
                  <span className="mint-badge is-planned">{r.statusLabel}</span>
                )}
              </td>

              {/* ★ リクエスト者（mints.createdByName） */}
              <td>{requesterName}</td>

              {/* ★ ミント日時（mintedAt） */}
              <td>{mintedAtLabel}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}

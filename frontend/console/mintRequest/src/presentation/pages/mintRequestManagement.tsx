// frontend/mintRequest/src/presentation/pages/mintRequestManagement.tsx

import React from "react";
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
        {rows.map((r) => (
          <tr
            key={r.id}
            onClick={() => handleRowClick(r.id)}
            style={{ cursor: "pointer" }}
            tabIndex={0}
            onKeyDown={(e) => handleRowKeyDown(e, r.id)}
            aria-label={`ミント申請 ${r.productName} の詳細へ`}
          >
            {/* ★ ミント申請ID列は削除（id は内部的にのみ使用） */}
            <td>
              <span className="lp-brand-pill">{r.tokenBlueprintId}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{r.productName}</span>
            </td>
            <td>{r.mintQuantity}</td>
            {/* ★ 生産量列（Mint数量の右隣り） */}
            <td>{r.productionQuantity ?? "-"}</td>
            <td>
              {r.status === "minted" ? (
                <span className="mint-badge is-done">{r.statusLabel}</span>
              ) : r.status === "requested" ? (
                <span className="mint-badge is-requested">
                  {r.statusLabel}
                </span>
              ) : (
                <span className="mint-badge is-planned">{r.statusLabel}</span>
              )}
            </td>
            <td>{r.requestedBy ?? "-"}</td>
            {/* ★ リクエスト日時列は削除 */}
            <td>{r.mintedAt ?? "-"}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}

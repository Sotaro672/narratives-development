// frontend/console/shell/src/pages/mintManagement.tsx

import type { KeyboardEvent } from "react";

import { useMintRequestManagement } from "../features/mint/presentation/hook/useMintRequestManagement";
import List from "../layout/List/List";
import { Badge, type BadgeVariant } from "../shared/ui/badge";
import { TableCell, TableRow } from "../shared/ui/table";

function getMintStatusBadgeVariant(status: string): BadgeVariant {
  switch (status) {
    case "minted":
      return "danger";
    case "minting":
      return "info";
    case "requested":
      return "active";
    default:
      return "default";
  }
}

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
          const requesterName = row.createdByName ?? "-";
          const mintedAtLabel =
            row.status === "minted"
              ? row.mintedAt ?? "-"
              : "-";
          const tokenLabel =
            row.tokenName ??
            row.tokenBlueprintId ??
            "-";
          const productName = row.productName ?? "-";

          return (
            <TableRow
              key={row.productionId}
              onClick={() => handleRowClick(row.productionId)}
              className="cursor-pointer"
              tabIndex={0}
              onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) =>
                handleRowKeyDown(
                  event,
                  row.productionId,
                )
              }
              aria-label={`ミント申請 ${productName} の詳細へ`}
            >
              <TableCell>
                <span className="truncate">
                  {tokenLabel}
                </span>
              </TableCell>

              <TableCell>
                <span className="truncate">
                  {productName}
                </span>
              </TableCell>

              <TableCell>{row.mintQuantity}</TableCell>
              <TableCell>{row.productionQuantity}</TableCell>

              <TableCell>
                <Badge variant={getMintStatusBadgeVariant(row.status)}>
                  {row.statusLabel}
                </Badge>
              </TableCell>

              <TableCell>{requesterName}</TableCell>
              <TableCell>{mintedAtLabel}</TableCell>
            </TableRow>
          );
        })}
      </List>
    </div>
  );
}
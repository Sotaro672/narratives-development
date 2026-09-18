// frontend/console/shell/src/pages/mintManagement.tsx

import List from "../layout/List/List";
import { Badge, type BadgeVariant } from "../shared/ui/badge";
import { useMintRequestManagement } from "../features/mint/presentation/hook/useMintRequestManagement";

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

          const productName =
            row.productName ?? "-";

          return (
            <tr
              key={row.productionId}
              onClick={() => handleRowClick(row.productionId)}
              className="cursor-pointer"
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

              <td>{row.mintQuantity}</td>
              <td>{row.productionQuantity}</td>

              <td>
                <Badge variant={getMintStatusBadgeVariant(row.status)}>
                  {row.statusLabel}
                </Badge>
              </td>

              <td>{requesterName}</td>
              <td>{mintedAtLabel}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}
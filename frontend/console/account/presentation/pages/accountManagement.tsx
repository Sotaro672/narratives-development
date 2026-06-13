// frontend/account/src/presentation/pages/accountManagement.tsx

import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";
import "../styles/account.css";

// Lucide型エラー対策
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

export default function AccountManagementPage() {
  const headers: React.ReactNode[] = [
    "口座ID",
    "会員ID",
    <>
      <span className="inline-flex items-center gap-2">
        銀行名
        <button className="lp-th-filter" aria-label="銀行名を絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "支店名",
    "口座番号",
    "種別",
    "通貨",
    <>
      <span className="inline-flex items-center gap-2">
        ステータス
        <button className="lp-th-filter" aria-label="ステータスを絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "登録日",
  ];

  return (
    <div className="p-0">
      <List
        title="口座管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton={false}
      >
        <tr>
          <td colSpan={9}>
            <div className="account-empty">
              試作品では口座管理機能は未実装です。
            </div>
          </td>
        </tr>
      </List>
    </div>
  );
}
// frontend/inquiry/src/presentation/pages/InquiryManagement.tsx

import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import "../styles/inquiry.css";

export default function InquiryManagementPage() {
  return (
    <div className="p-0">
      <List
        title="問い合わせ管理"
        headerCells={[
          "問い合わせID",
          "件名",
          "ユーザー(ID)",
          "ステータス",
          "タイプ",
          "担当者 (memberId)",
          "問い合わせ日",
          "最終更新日",
        ]}
        showCreateButton={false}
        showResetButton={false}
      >
        <tr>
          <td colSpan={8}>
            <div className="inq__empty">
              試作品では問い合わせ管理機能は未実装です。
            </div>
          </td>
        </tr>
      </List>
    </div>
  );
}
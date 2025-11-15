// frontend/member/src/presentation/pages/memberManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../../hooks/useMemberList";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  // データ取得はフックに委譲（companyId の補完や正規化もそちらで実施）
  const { members, loading, error, reload } = useMemberList();

  const goDetail = (id: string) => {
    if (!id) return;
    navigate(`/member/${encodeURIComponent(id)}`);
  };

  const ymd = (createdAt: any): string => {
    if (!createdAt) return "";
    // Firestore Timestamp (toDate) / seconds / ISO文字列対応の簡易表示
    if (typeof createdAt === "object" && createdAt !== null) {
      if (typeof (createdAt as any).toDate === "function") {
        return (createdAt as any).toDate().toISOString().slice(0, 10).replace(/-/g, "/");
      }
      if (typeof (createdAt as any).seconds === "number") {
        return new Date((createdAt as any).seconds * 1000).toISOString().slice(0, 10).replace(/-/g, "/");
      }
    }
    if (typeof createdAt === "string") {
      return createdAt.slice(0, 10).replace(/-/g, "/");
    }
    return "";
    // ※ 体裁整形のみ。ロジックはフック側に集約済み。
  };

  if (loading) return <div className="p-4">読み込み中...</div>;
  if (error)
    return (
      <div className="p-4 text-red-500">
        データ取得エラー: {error.message}
      </div>
    );

  return (
    <div className="p-0">
      <List
        title="メンバー管理"
        headerCells={[
          "氏名",
          "メールアドレス",
          "所属ブランド",
          "権限数",
          "登録日",
        ]}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        onCreate={() => navigate("/member/create")}
        onReset={() => {
          reload();
        }}
      >
        {members.map((m) => {
          const name =
            `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim() ||
            (m.email ?? "") ||
            m.id;

          const brands = m.assignedBrands ?? [];
          const permissionCount = m.permissions?.length ?? 0;
          const registeredAt = ymd((m as any).createdAt);

          return (
            <tr
              key={m.id}
              role="button"
              tabIndex={0}
              className="cursor-pointer"
              onClick={() => goDetail(m.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  goDetail(m.id);
                }
              }}
            >
              <td>{name}</td>
              <td>{m.email ?? ""}</td>
              <td>
                {brands.map((b) => (
                  <span key={b} className="lp-brand-pill mm-brand-tag">
                    {b}
                  </span>
                ))}
              </td>
              <td>{permissionCount}</td>
              <td>{registeredAt}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}

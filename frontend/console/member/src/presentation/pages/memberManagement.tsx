// frontend/member/src/presentation/pages/memberManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../hooks/useMemberList";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  // メンバー一覧 + 氏名解決関数 + brandId→brandName マップを hook から取得
  const {
    members,
    loading,
    error,
    reload,
    getNameLastFirstByID,
    brandMap,
  } = useMemberList();

  // 氏名キャッシュ（画面側で保持）
  const [resolvedNames, setResolvedNames] = React.useState<
    Record<string, string>
  >({});

  // -------------------------
  //  氏名補完
  // -------------------------
  React.useEffect(() => {
    let disposed = false;

    (async () => {
      const entries = await Promise.all(
        members.map(async (m) => {
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          if (inline) return [m.id, inline] as const;

          const resolved = await getNameLastFirstByID(m.id);
          return [m.id, resolved] as const;
        }),
      );

      if (!disposed) {
        const next: Record<string, string> = {};
        for (const [id, name] of entries) {
          if (name) next[id] = name;
        }
        setResolvedNames(next);
      }
    })();

    return () => {
      disposed = true;
    };
  }, [members, getNameLastFirstByID]);

  const goDetail = (id: string) => {
    if (!id) return;
    navigate(`/member/${encodeURIComponent(id)}`);
  };

  const ymd = (createdAt: any): string => {
    if (!createdAt) return "";
    if (typeof createdAt === "object" && createdAt !== null) {
      if (typeof (createdAt as any).toDate === "function") {
        return (createdAt as any)
          .toDate()
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
      if (typeof (createdAt as any).seconds === "number") {
        return new Date((createdAt as any).seconds * 1000)
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
    }
    if (typeof createdAt === "string") {
      return createdAt.slice(0, 10).replace(/-/g, "/");
    }
    return "";
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
        headerCells={["氏名", "メールアドレス", "所属ブランド", "権限数", "登録日"]}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        onCreate={() => navigate("/member/create")}
        onReset={() => reload()}
      >
        {members.map((m) => {
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          const name =
            resolvedNames[m.id] || inline || (m.email ?? "") || m.id;

          const assigned = m.assignedBrands ?? [];

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
              <td>{name || "招待中"}</td>
              <td>{m.email ?? ""}</td>
              <td>
                {assigned.map((brandId) => {
                  const label = brandMap[brandId] ?? brandId;
                  return (
                    <span
                      key={brandId}
                      className="lp-brand-pill mm-brand-tag"
                    >
                      {label}
                    </span>
                  );
                })}
              </td>
              <td>{m.permissions?.length ?? 0}</td>
              <td>{ymd((m as any).createdAt)}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}

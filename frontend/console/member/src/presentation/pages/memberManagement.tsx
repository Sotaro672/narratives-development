// frontend/member/src/presentation/pages/memberManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../../hooks/useMemberList";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  // フックから ID→氏名解決関数も受け取る
  const { members, loading, error, reload, getNameLastFirstByID } = useMemberList();

  // 非同期に解決した氏名を保持（id -> "姓 名"）
  const [resolvedNames, setResolvedNames] = React.useState<Record<string, string>>({});

  // 氏名が空の行だけ ID→氏名を解決してキャッシュ
  React.useEffect(() => {
    let disposed = false;

    (async () => {
      const entries = await Promise.all(
        members.map(async (m) => {
          const immediate = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          if (immediate) return [m.id, immediate] as const;

          // 一覧に名前が無い場合だけバックエンドへ（useMemberList 側でキャッシュあり）
          const resolved = await getNameLastFirstByID(m.id);
          return [m.id, resolved] as const;
        }),
      );

      if (!disposed) {
        const next: Record<string, string> = {};
        for (const [id, name] of entries) {
          if (name) next[id] = name; // 空は保存しない（招待中やメールでフォールバック）
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
        return (createdAt as any).toDate().toISOString().slice(0, 10).replace(/-/g, "/");
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
        onReset={() => {
          reload();
        }}
      >
        {members.map((m) => {
          // 1) 解決済み氏名 2) その場の氏名 3) メール 4) ID の順でフォールバック
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          const name =
            resolvedNames[m.id] || inline || (m.email ?? "") || m.id;

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
              <td>{name || "招待中"}</td>
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

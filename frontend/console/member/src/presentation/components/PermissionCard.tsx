// frontend/console/member/src/presentation/components/PermissionCard.tsx

import * as React from "react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import type { PermissionCategory } from "../../../../shell/src/shared/types/permission";

type PermissionCardProps = {
  /** メンバーに付与されている Permission.Name の配列 */
  permissions: string[];

  /**
   * useMemberDetail で Category -> Permission.Name[] にグルーピングした結果
   * 例: { brand: ["brand.read", "brand.list"], member: ["member.read"] }
   */
  groupedByCategory: Partial<Record<PermissionCategory, string[]>>;

  /** 権限カタログ取得のローディング状態（useMemberDetail から渡す） */
  loading: boolean;
};

export function PermissionCard({
  permissions,
  groupedByCategory,
  loading,
}: PermissionCardProps) {
  const categories = Object.keys(groupedByCategory) as PermissionCategory[];

  const hasGrouped = categories.length > 0 && permissions.length > 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle>権限</CardTitle>
      </CardHeader>
      <CardContent>
        {permissions.length === 0 ? (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            権限は未設定です。
          </p>
        ) : loading && !hasGrouped ? (
          <p className="text-sm text-[hsl(var(--muted-foreground))]">
            権限情報を読み込み中です…
          </p>
        ) : hasGrouped ? (
          <div className="space-y-4">
            {categories.map((cat, index) => {
              const list = groupedByCategory[cat];
              if (!list || list.length === 0) return null;

              return (
                <div key={cat}>
                  {/* ---- 区切りバー：最初のカテゴリ以外に入れる ---- */}
                  {index > 0 && <div className="permission-category-divider" />}

                  {/* ▼ 親要素: Category */}
                  <div className="text-xs font-semibold text-slate-500 mb-1">
                    {cat}
                  </div>

                  {/* ▼ 子要素 */}
                  <ul className="text-sm space-y-1 ml-3 list-disc">
                    {list.map((perm) => (
                      <li key={`${cat}:${perm}`}>{perm}</li>
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>
        ) : (
          // フォールバック（カテゴリ情報が取得できなかった場合）
          <ul className="text-sm space-y-1">
            {permissions.map((perm) => (
              <li key={perm}>{perm}</li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

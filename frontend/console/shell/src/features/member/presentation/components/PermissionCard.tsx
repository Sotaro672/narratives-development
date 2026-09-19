// frontend/console/member/src/presentation/components/PermissionCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import type { PermissionCategory } from "../../../../shared/types/permission";

import "../../../../shell/src/styles/permission.css";

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
  const categories = Object.keys(
    groupedByCategory,
  ) as PermissionCategory[];

  const hasGrouped =
    categories.length > 0 &&
    permissions.length > 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle>権限</CardTitle>
      </CardHeader>

      <CardContent>
        {permissions.length === 0 ? (
          <p className="permission-card__message">
            権限は未設定です。
          </p>
        ) : loading && !hasGrouped ? (
          <p className="permission-card__message">
            権限情報を読み込み中です…
          </p>
        ) : hasGrouped ? (
          <div className="permission-card__groups">
            {categories.map((cat, index) => {
              const list = groupedByCategory[cat];

              if (!list || list.length === 0) {
                return null;
              }

              return (
                <div key={cat}>
                  {index > 0 ? (
                    <div className="permission-card__divider" />
                  ) : null}

                  <div className="permission-card__category">
                    {cat}
                  </div>

                  <ul className="permission-card__list permission-card__list--indented">
                    {list.map((perm) => (
                      <li key={`${cat}:${perm}`}>
                        {perm}
                      </li>
                    ))}
                  </ul>
                </div>
              );
            })}
          </div>
        ) : (
          <ul className="permission-card__list permission-card__list--fallback">
            {permissions.map((perm) => (
              <li key={perm}>
                {perm}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
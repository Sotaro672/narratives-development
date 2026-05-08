// frontend/console/member/src/presentation/components/PermissionSelect.tsx
import * as React from "react";

import type {
  Permission,
  PermissionCategory,
} from "../../../../shell/src/shared/types/permission";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import { Button } from "../../../../shell/src/shared/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Checkbox } from "../../../../shell/src/shared/ui/checkbox";
import { Badge } from "../../../../shell/src/shared/ui/badge";

type PermissionCategoryView = {
  key: PermissionCategory;
  count: number;
  permissions: Permission[];
};

export type PermissionSelectProps = {
  category: PermissionCategory;
  setCategory: (c: PermissionCategory) => void;

  permissionCategories: PermissionCategoryView[];
  permissionCategoryList: PermissionCategory[];

  selectedPermIds: Set<string>;
  setSelectedPermIds: React.Dispatch<React.SetStateAction<Set<string>>>;
};

const categoryLabel = (c?: string) => c ?? "";

// Popover を閉じるためのユーティリティ
const closePopover = () =>
  document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));

export function PermissionSelect({
  category,
  setCategory,
  permissionCategories,
  permissionCategoryList,
  selectedPermIds,
  setSelectedPermIds,
}: PermissionSelectProps) {
  // ========= 権限カテゴリ ==========
  const selectedCategory = React.useMemo(
    () => permissionCategories.find((x) => x.key === category),
    [permissionCategories, category],
  );
  const currentPerms = selectedCategory?.permissions ?? [];

  const togglePerm = (permId: string, checked: boolean) => {
    setSelectedPermIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(permId);
      else next.delete(permId);
      return next;
    });
  };

  const allSelectedInCategory =
    currentPerms.length > 0 &&
    currentPerms.every((p) => selectedPermIds.has((p as any).id));

  const toggleAllInCategory = (checked: boolean) => {
    if (currentPerms.length === 0) return;
    setSelectedPermIds((prev) => {
      const next = new Set(prev);
      if (checked)
        currentPerms.forEach((p) => next.add((p as any).id as string));
      else currentPerms.forEach((p) => next.delete((p as any).id as string));
      return next;
    });
  };

  const allPerms = React.useMemo(
    () =>
      permissionCategories.flatMap((c) => c.permissions) as (Permission & {
        id: string;
      })[],
    [permissionCategories],
  );

  const selectedPerms = React.useMemo(
    () => allPerms.filter((p) => selectedPermIds.has(p.id)),
    [allPerms, selectedPermIds],
  );

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-start">
      {/* 役割選択 */}
      <div>
        <label className="block text-sm text-slate-300 mb-1">
          役割（必須）
        </label>
        <Popover>
          <PopoverTrigger>
            <Button
              type="button"
              variant="outline"
              className="role-button w-full justify-start text-left"
            >
              {category ? (
                categoryLabel(category)
              ) : (
                <span className="text-slate-400">役割を選択</span>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent className="text-sm max-h-[320px] overflow-y-auto">
            <div className="flex flex-col gap-1">
              {permissionCategoryList.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => {
                    setCategory(c);
                    closePopover();
                  }}
                  className={
                    "w-full text-left px-3 py-2 rounded hover:bg-slate-100 " +
                    (category === c ? "bg-slate-100 font-semibold" : "")
                  }
                >
                  {categoryLabel(c)}
                </button>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      </div>

      {/* 権限一覧 + 選択済みバッジ */}
      <div>
        <Card>
          <CardHeader>
            <Checkbox
              id="category-select-all"
              checked={allSelectedInCategory}
              onCheckedChange={(v) => toggleAllInCategory(!!v)}
            />
            <CardTitle style={{ marginLeft: 8 }}>
              権限一覧（{categoryLabel(category)}）
            </CardTitle>
          </CardHeader>
          <CardContent>
            {currentPerms.length === 0 ? (
              <p className="text-sm text-[hsl(var(--muted-foreground))]">
                この役割に紐づく権限はありません。
              </p>
            ) : (
              <ul className="text-sm space-y-2">
                {currentPerms.map((perm: any) => {
                  const checked = selectedPermIds.has(perm.id);
                  const inputId = `perm_${perm.id}`;
                  return (
                    <li key={perm.id} className="flex items-start gap-2">
                      <Checkbox
                        id={inputId}
                        checked={checked}
                        onCheckedChange={(v) => togglePerm(perm.id, !!v)}
                      />
                      <label
                        id={inputId}
                        className="cursor-pointer select-none"
                        onClick={() => togglePerm(perm.id, !checked)}
                      >
                        <span className="font-medium">{perm.name}</span>
                        <span className="text-[hsl(var(--muted-foreground))]">
                          {" — "}
                          {perm.description}
                        </span>
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
          </CardContent>
        </Card>

        {/* 選択済み権限バッジ */}
        <div className="mt-3 flex flex-wrap gap-2">
          {selectedPerms.length === 0 ? (
            <span className="text-xs text-[hsl(var(--muted-foreground))]">
              権限を選択するとここに表示されます。
            </span>
          ) : (
            selectedPerms.map((perm) => (
              <Badge key={`badge_${perm.id}`} variant="secondary">
                {perm.name}
              </Badge>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

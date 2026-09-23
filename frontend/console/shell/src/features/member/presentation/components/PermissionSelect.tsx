// frontend/console/shell/src/features/member/presentation/components/PermissionSelect.tsx

import * as React from "react";

import type {
  Permission,
  PermissionCategory,
} from "../../../../shared/types/permission";

import { Badge } from "../../../../shared/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Checkbox } from "../../../../shared/ui/checkbox";
import { Select } from "../../../../shared/ui/select";

import "../../../../styles/permission.css";

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

export function PermissionSelect({
  category,
  setCategory,
  permissionCategories,
  permissionCategoryList,
  selectedPermIds,
  setSelectedPermIds,
}: PermissionSelectProps) {
  const selectedCategory = React.useMemo(
    () => permissionCategories.find((item) => item.key === category),
    [permissionCategories, category],
  );

  const currentPerms = selectedCategory?.permissions ?? [];

  const categoryOptions = React.useMemo(
    () =>
      permissionCategoryList.map((item) => ({
        value: item,
        label: categoryLabel(item),
      })),
    [permissionCategoryList],
  );

  const togglePerm = (permId: string, checked: boolean) => {
    setSelectedPermIds((prev) => {
      const next = new Set(prev);

      if (checked) {
        next.add(permId);
      } else {
        next.delete(permId);
      }

      return next;
    });
  };

  const allSelectedInCategory =
    currentPerms.length > 0 &&
    currentPerms.every((permission) =>
      selectedPermIds.has((permission as any).id),
    );

  const toggleAllInCategory = (checked: boolean) => {
    if (currentPerms.length === 0) {
      return;
    }

    setSelectedPermIds((prev) => {
      const next = new Set(prev);

      if (checked) {
        currentPerms.forEach((permission) =>
          next.add((permission as any).id as string),
        );
      } else {
        currentPerms.forEach((permission) =>
          next.delete((permission as any).id as string),
        );
      }

      return next;
    });
  };

  const allPerms = React.useMemo(
    () =>
      permissionCategories.flatMap((item) => item.permissions) as (Permission & {
        id: string;
      })[],
    [permissionCategories],
  );

  const selectedPerms = React.useMemo(
    () => allPerms.filter((permission) => selectedPermIds.has(permission.id)),
    [allPerms, selectedPermIds],
  );

  return (
    <div className="permission-select">
      <Select
        label="役割（必須）"
        options={categoryOptions}
        value={category}
        placeholder="役割を選択"
        emptyText="選択可能な役割がありません。"
        onChange={(value) => {
          setCategory(value as PermissionCategory);
        }}
      />

      <div>
        <Card>
          <CardHeader>
            <Checkbox
              id="category-select-all"
              checked={allSelectedInCategory}
              onCheckedChange={(value) => toggleAllInCategory(!!value)}
            />

            <CardTitle className="permission-select__card-title">
              権限一覧（{categoryLabel(category)}）
            </CardTitle>
          </CardHeader>

          <CardContent>
            {currentPerms.length === 0 ? (
              <p className="permission-select__empty">
                この役割に紐づく権限はありません。
              </p>
            ) : (
              <ul className="permission-select__list">
                {currentPerms.map((perm: any) => {
                  const checked = selectedPermIds.has(perm.id);
                  const inputId = `perm_${perm.id}`;

                  return (
                    <li
                      key={perm.id}
                      className="permission-select__item"
                    >
                      <Checkbox
                        id={inputId}
                        checked={checked}
                        onCheckedChange={(value) =>
                          togglePerm(perm.id, !!value)
                        }
                      />

                      <label
                        htmlFor={inputId}
                        className="permission-select__item-label"
                      >
                        <span className="permission-select__permission-name">
                          {perm.name}
                        </span>

                        <span className="permission-select__permission-description">
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

        <div className="permission-select__badges">
          {selectedPerms.length === 0 ? (
            <span className="permission-select__hint">
              権限を選択するとここに表示されます。
            </span>
          ) : (
            selectedPerms.map((perm) => (
              <Badge
                key={`badge_${perm.id}`}
                variant="secondary"
              >
                {perm.name}
              </Badge>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
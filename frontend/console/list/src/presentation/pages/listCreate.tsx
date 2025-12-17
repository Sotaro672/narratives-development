// frontend/console/list/src/presentation/pages/listCreate.tsx
import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

// Table UI
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shell/src/shared/ui/table";

import { useListCreate, type CandidateRow } from "../hook/useListCreate";

export default function ListCreate() {
  const {
    onBack,
    onCreate,

    product,
    setProduct,
    brand,
    setBrand,
    token,
    setToken,
    stock,
    setStock,
    manager,
    setManager,
    status,
    setStatus,

    assigneeName,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee,

    selectedBrand,
    brandOptions,
    selectBrand,

    candidateRows,
    selectedCandidateId,
    selectCandidateById,
  } = useListCreate();

  return (
    <PageStyle
      layout="grid-2"
      title="出品の作成"
      onBack={onBack}
      onSave={onCreate}
    >
      {/* ========== 左ペイン（出品作成フォーム） ========== */}
      <div className="list-create-form">
        <h2 className="section-title">出品情報</h2>

        <div className="form-group">
          <label>プロダクト名</label>
          <input
            type="text"
            value={product}
            onChange={(e) => setProduct(e.target.value)}
            placeholder="例: シルクブラウス プレミアムライン"
          />
        </div>

        <div className="form-group">
          <label>ブランド</label>
          <input
            type="text"
            value={brand}
            onChange={(e) => setBrand(e.target.value)}
            placeholder="例: LUMINA Fashion"
          />
          <div className="text-xs text-gray-500 mt-1">
            ※ 右カラムの「ブランド選択」と同期します
          </div>
        </div>

        <div className="form-group">
          <label>トークン</label>
          <input
            type="text"
            value={token}
            onChange={(e) => setToken(e.target.value)}
            placeholder="例: LUMINA VIP Token"
          />
        </div>

        <div className="form-group">
          <label>在庫数</label>
          <input
            type="number"
            value={stock}
            onChange={(e) =>
              setStock(e.target.value === "" ? "" : Number(e.target.value))
            }
            placeholder="例: 200"
          />
        </div>

        <div className="form-group">
          <label>担当者</label>
          <input
            type="text"
            value={manager}
            onChange={(e) => setManager(e.target.value)}
            placeholder="例: 山田 太郎"
          />
        </div>

        <div className="form-group">
          <label>ステータス</label>
          <select
            value={status}
            onChange={(e) =>
              setStatus(e.target.value as "出品中" | "停止中" | "")
            }
          >
            <option value="">選択してください</option>
            <option value="出品中">出品中</option>
            <option value="停止中">停止中</option>
          </select>
        </div>
      </div>

      {/* ========== 右ペイン ========== */}
      <div className="space-y-4">
        {/* 管理情報カード */}
        <AdminCard
          mode="edit"
          title="管理情報"
          assigneeName={assigneeName}
          assigneeCandidates={assigneeOptions}
          loadingMembers={loadingMembers}
          onSelectAssignee={onSelectAssignee}
        />

        {/* 対象一覧（ヘッダー左にブランド選択ボタン） */}
        <Card className="pb-select">
          <CardHeader>
            <CardTitle>対象一覧</CardTitle>
          </CardHeader>

          <CardContent>
            <Table className="border rounded">
              <TableHeader>
                <TableRow>
                  <TableHead>
                    <div className="flex items-center justify-start gap-2">
                      <Popover>
                        <PopoverTrigger>
                          <div className="pb-select__trigger">
                            {selectedBrand || "ブランド選択"}
                          </div>
                        </PopoverTrigger>

                        <PopoverContent>
                          <div className="pb-select__list">
                            {brandOptions.map((b: string) => (
                              <button
                                key={b}
                                className={
                                  "pb-select__row" +
                                  (selectedBrand === b ? " is-active" : "")
                                }
                                onClick={() => selectBrand(b)}
                              >
                                {b}
                              </button>
                            ))}

                            {brandOptions.length === 0 && (
                              <div className="pb-select__empty">
                                ブランドが登録されていません。
                              </div>
                            )}
                          </div>
                        </PopoverContent>
                      </Popover>
                    </div>
                  </TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {candidateRows.map((row: CandidateRow) => (
                  <TableRow
                    key={row.id}
                    className={
                      "cursor-pointer hover:bg-blue-50" +
                      (selectedCandidateId === row.id ? " bg-blue-100" : "")
                    }
                    onClick={() => selectCandidateById(row.id)}
                  >
                    <TableCell>{row.name}</TableCell>
                  </TableRow>
                ))}

                {candidateRows.length === 0 && (
                  <TableRow>
                    <TableCell className="text-center text-gray-500">
                      対象がありません
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>

            {selectedCandidateId && (
              <div className="mt-2 text-xs text-gray-500">
                選択中: {selectedCandidateId}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}

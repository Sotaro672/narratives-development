// frontend/admin/src/App.tsx
import * as React from "react";
import AdminCard from "./pages/AdminCard"; // 既に作成済みのカード

export default function App() {
  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold mb-4">Admin</h1>
      <AdminCard
        assigneeName="佐藤 美咲"
        createdByName="山田 太郎"
        createdAt="2024/1/20"
        onEditAssignee={() => console.log("edit")}
      />
    </div>
  );
}

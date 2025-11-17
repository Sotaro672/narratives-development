import { useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";

export function useBrandCreate() {
  const navigate = useNavigate();

  // フォーム状態
  const [brandName, setBrandName] = useState("");
  const [brandCode, setBrandCode] = useState("");
  const [category, setCategory] = useState("ファッション");
  const [description, setDescription] = useState("");

  // 戻る処理
  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 保存処理（モック）
  const handleSave = useCallback(() => {
    console.log("保存:", {
      brandName,
      brandCode,
      category,
      description,
    });
    alert("ブランド情報を保存しました（モック）");
  }, [brandName, brandCode, category, description]);

  return {
    brandName,
    setBrandName,
    brandCode,
    setBrandCode,
    category,
    setCategory,
    description,
    setDescription,
    handleBack,
    handleSave,
  };
}

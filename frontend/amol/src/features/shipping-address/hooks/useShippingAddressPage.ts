//frontend\amol\src\features\shipping-address\hooks\useShippingAddressPage.ts
import {
  FormEvent,
  useEffect,
  useState,
  type ChangeEvent,
} from "react";
import { getAuth } from "firebase/auth";
import { useNavigate } from "react-router-dom";

import {
  fetchShippingAddressPageInitialData,
  saveShippingAddress,
  saveUserProfile,
} from "../api/shippingAddressApi";
import type {
  ShippingAddress,
  ShippingAddressFormValues,
  ShippingAddressPageMode,
} from "../types";
import {
  getShippingAddressId,
  isUserProfile,
} from "../utils/zipCode";
import { useZipCodeAddressLookup } from "./useZipCodeAddressLookup";

const initialForm: ShippingAddressFormValues = {
  lastName: "",
  firstName: "",
  lastNameKana: "",
  firstNameKana: "",
  zipCode: "",
  state: "",
  city: "",
  street: "",
  street2: "",
};

export function useShippingAddressPage() {
  const navigate = useNavigate();
  const auth = getAuth();

  const [shippingAddress, setShippingAddress] =
    useState<ShippingAddress | null>(null);

  const [pageMode, setPageMode] =
    useState<ShippingAddressPageMode>("create");

  const [form, setForm] =
    useState<ShippingAddressFormValues>(initialForm);

  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  const shippingAddressId = getShippingAddressId(shippingAddress);
  const isEditMode = pageMode === "edit" && Boolean(shippingAddressId);

  const {
    isLookingUpAddress,
    zipCodeError,
    resetZipCodeLookup,
  } = useZipCodeAddressLookup({
    zipCode: form.zipCode,
    setForm,
  });

  const actionButtonLabel = isSaving
    ? "保存中..."
    : isEditMode
      ? "更新"
      : "登録";

  const actionButtonDisabled = isLoading || isSaving;

  useEffect(() => {
    const fetchInitialData = async () => {
      const currentUser = auth.currentUser;

      if (!currentUser) {
        navigate("/signin", { replace: true });
        return;
      }

      try {
        const backendUrl = import.meta.env.VITE_API_BASE_URL;

        if (!backendUrl) {
          throw new Error("VITE_API_BASE_URL が設定されていません。");
        }

        const idToken = await currentUser.getIdToken(true);

        const { userProfile, shippingAddresses } =
          await fetchShippingAddressPageInitialData({
            backendUrl,
            idToken,
          });

        const firstAddress = shippingAddresses[0] || null;

        if (firstAddress) {
          setShippingAddress(firstAddress);
          setPageMode("edit");
        } else {
          setShippingAddress(null);
          setPageMode("create");
        }

        setForm((current) => ({
          ...current,
          lastName: isUserProfile(userProfile)
            ? userProfile.last_name || ""
            : "",
          firstName: isUserProfile(userProfile)
            ? userProfile.first_name || ""
            : "",
          lastNameKana: isUserProfile(userProfile)
            ? userProfile.last_name_kana || ""
            : "",
          firstNameKana: isUserProfile(userProfile)
            ? userProfile.first_name_kana || ""
            : "",
          zipCode: firstAddress?.zipCode || "",
          state: firstAddress?.state || "",
          city: firstAddress?.city || "",
          street: firstAddress?.street || "",
          street2: firstAddress?.street2 || "",
        }));
      } catch (error) {
        console.error(error);

        if (error instanceof Error) {
          window.alert(error.message);
        } else {
          window.alert("配送先情報の取得に失敗しました。");
        }
      } finally {
        setIsLoading(false);
      }
    };

    void fetchInitialData();
  }, [auth, navigate]);

  const handleChange =
    (name: keyof ShippingAddressFormValues) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      if (name === "zipCode") {
        resetZipCodeLookup();
      }

      setForm((current) => ({
        ...current,
        [name]: event.target.value,
      }));
    };

  const handleSave = async () => {
    const currentUser = auth.currentUser;

    if (!currentUser) {
      window.alert("ログイン情報を確認できませんでした。");
      return;
    }

    try {
      setIsSaving(true);

      const backendUrl = import.meta.env.VITE_API_BASE_URL;

      if (!backendUrl) {
        throw new Error("VITE_API_BASE_URL が設定されていません。");
      }

      const idToken = await currentUser.getIdToken(true);

      await saveUserProfile({
        backendUrl,
        idToken,
        payload: {
          last_name: form.lastName,
          first_name: form.firstName,
          last_name_kana: form.lastNameKana,
          first_name_kana: form.firstNameKana,
        },
      });

      const savedShippingAddress = await saveShippingAddress({
        backendUrl,
        idToken,
        isEditMode,
        shippingAddressId,
        payload: {
          zipCode: form.zipCode,
          state: form.state,
          city: form.city,
          street: form.street,
          street2: form.street2,
          country: "JP",
        },
      });

      if (savedShippingAddress) {
        setShippingAddress(savedShippingAddress);
        setPageMode("edit");

        setForm((current) => ({
          ...current,
          zipCode: savedShippingAddress.zipCode || "",
          state: savedShippingAddress.state || "",
          city: savedShippingAddress.city || "",
          street: savedShippingAddress.street || "",
          street2: savedShippingAddress.street2 || "",
        }));
      }

      window.alert(
        isEditMode
          ? "配送先情報を更新しました。"
          : "配送先情報を登録しました。"
      );
    } catch (error) {
      console.error(error);

      if (error instanceof Error) {
        window.alert(error.message);
      } else {
        window.alert(
          isEditMode
            ? "配送先情報の更新に失敗しました。"
            : "配送先情報の登録に失敗しました。"
        );
      }
    } finally {
      setIsSaving(false);
    }
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    void handleSave();
  };

  return {
    form,
    isLoading,
    isSaving,
    isEditMode,
    isLookingUpAddress,
    zipCodeError,
    actionButtonLabel,
    actionButtonDisabled,
    handleChange,
    handleSave,
    handleSubmit,
  };
}
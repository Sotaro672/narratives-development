// frontend/amol/src/features/avatar/services/avatarCreateService.ts

import type { Auth } from "firebase/auth";
import { getDownloadURL, ref, uploadBytes } from "firebase/storage";

import { storage } from "../../../lib/firebase";
import { createAvatar, getMyAvatar, updateAvatar } from "../api/avatarApi";
import type {
  AvatarCreateResult,
  AvatarUpdateResult,
  MyAvatarResponse,
  PickIconResult,
} from "../../shared/types/avatar";

type AvatarCreateServiceParams = {
  auth: Auth;
};

type SaveAvatarParams = {
  avatarNameRaw: string;
  profileRaw: string;
  externalLinkRaw: string;
  iconFile: File | null;
};

type UpdateAvatarParams = SaveAvatarParams & {
  avatarId: string;
};

export class AvatarCreateService {
  private readonly auth: Auth;

  constructor({ auth }: AvatarCreateServiceParams) {
    this.auth = auth;
  }

  backTo(from: string | null): string {
    return from || "/lists";
  }

  isValidUrlOrEmpty(value: string): boolean {
    if (!value) return true;

    try {
      const url = new URL(value);
      return (url.protocol === "http:" || url.protocol === "https:") && !!url.host;
    } catch {
      return false;
    }
  }

  pickIconWeb(file: File | null): PickIconResult | null {
    if (!file) return null;

    if (!file.type.startsWith("image/")) {
      return {
        file: null,
        fileName: null,
        mimeType: null,
        previewUrl: null,
        error: "画像ファイルを選択してください。",
      };
    }

    return {
      file,
      fileName: file.name || null,
      mimeType: file.type || null,
      previewUrl: URL.createObjectURL(file),
    };
  }

  private ensureMimeType(file: File): string {
    if (file.type) return file.type;

    const name = file.name.toLowerCase();

    if (name.endsWith(".png")) return "image/png";
    if (name.endsWith(".jpg") || name.endsWith(".jpeg")) return "image/jpeg";
    if (name.endsWith(".webp")) return "image/webp";
    if (name.endsWith(".gif")) return "image/gif";

    return "application/octet-stream";
  }

  private ensureSupportedImage(file: File): string {
    const mimeType = this.ensureMimeType(file).toLowerCase();

    switch (mimeType) {
      case "image/png":
      case "image/jpeg":
      case "image/jpg":
      case "image/webp":
      case "image/gif":
        return mimeType;
      default:
        throw new Error("対応していない画像形式です。png, jpg, webp, gif を選択してください。");
    }
  }

  private avatarIconStoragePath(avatarId: string): string {
    return `avatar-icons/${avatarId}/icon`;
  }

  private async uploadAvatarIconToFirebaseStorage({
    avatarId,
    iconFile,
  }: {
    avatarId: string;
    iconFile: File;
  }): Promise<string> {
    const mimeType = this.ensureSupportedImage(iconFile);
    const objectPath = this.avatarIconStoragePath(avatarId);
    const storageRef = ref(storage, objectPath);

    await uploadBytes(storageRef, iconFile, {
      contentType: mimeType,
      customMetadata: {
        avatarId,
        fileName: iconFile.name || "icon",
      },
    });

    return getDownloadURL(storageRef);
  }

  async fetchMine(): Promise<MyAvatarResponse | null> {
    return getMyAvatar();
  }

  async save({
    avatarNameRaw,
    profileRaw,
    externalLinkRaw,
    iconFile,
  }: SaveAvatarParams): Promise<AvatarCreateResult> {
    try {
      const user = this.auth.currentUser;

      if (!user) {
        return {
          ok: false,
          message: "サインインが必要です。",
        };
      }

      if (!user.uid) {
        return {
          ok: false,
          message: "userUid が取得できませんでした。",
        };
      }

      if (!avatarNameRaw) {
        return {
          ok: false,
          message: "アバター名を入力してください。",
        };
      }

      if (!this.isValidUrlOrEmpty(externalLinkRaw)) {
        return {
          ok: false,
          message: "外部リンクは http(s) のURLを入力してください。",
        };
      }

      const created = await createAvatar({
        payload: {
          userUid: user.uid,
          avatarName: avatarNameRaw,
          ...(profileRaw ? { profile: profileRaw } : {}),
          ...(externalLinkRaw ? { externalLink: externalLinkRaw } : {}),
        },
      });

      if (iconFile) {
        const avatarIcon = await this.uploadAvatarIconToFirebaseStorage({
          avatarId: created.avatarId,
          iconFile,
        });

        await updateAvatar({
          payload: {
            avatarName: created.avatarName,
            profile: created.profile ?? "",
            externalLink: created.externalLink ?? "",
            avatarIcon,
          },
        });
      }

      return {
        ok: true,
        message: "アバターを作成しました。",
        nextRoute: "/lists",
        createdAvatarId: created.avatarId,
      };
    } catch (error) {
      return {
        ok: false,
        message: error instanceof Error ? error.message : String(error),
      };
    }
  }

  async update({
    avatarId,
    avatarNameRaw,
    profileRaw,
    externalLinkRaw,
    iconFile,
  }: UpdateAvatarParams): Promise<AvatarUpdateResult> {
    try {
      if (!avatarNameRaw) {
        return {
          ok: false,
          message: "アバター名を入力してください。",
        };
      }

      if (!this.isValidUrlOrEmpty(externalLinkRaw)) {
        return {
          ok: false,
          message: "外部リンクは http(s) のURLを入力してください。",
        };
      }

      let avatarIcon: string | undefined;

      if (iconFile) {
        avatarIcon = await this.uploadAvatarIconToFirebaseStorage({
          avatarId,
          iconFile,
        });
      }

      const updated = await updateAvatar({
        payload: {
          avatarName: avatarNameRaw,
          profile: profileRaw,
          externalLink: externalLinkRaw,
          ...(avatarIcon ? { avatarIcon } : {}),
        },
      });

      return {
        ok: true,
        message: "アバターを保存しました。",
        avatarId: updated.avatarId,
      };
    } catch (error) {
      return {
        ok: false,
        message: error instanceof Error ? error.message : String(error),
      };
    }
  }
}
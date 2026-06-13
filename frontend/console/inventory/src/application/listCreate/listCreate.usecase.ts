// frontend/console/inventory/src/application/listCreate/listCreate.usecase.ts

import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";
import type { ListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.types";
import { mapListCreateDTO } from "../../infrastructure/http/listCreateRepositoryHTTP.mappers";

// Firebase Auth（uid 取得）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// list create (POST /lists)
import { createListHTTP } from "../../../../list/infrastructure/repository";
import type {
  CreateListInput,
  ListDTO,
} from "../../../../list/infrastructure/dto";

import type { ResolvedListCreateParams } from "./listCreate.types";
import { buildListCreateFetchInput } from "./listCreate.routing";
import {
  buildCreateListInput,
  validateCreateListInput,
} from "./listCreate.input";
import {
  uploadListImagesPolicyB,
  _internal_getListIdFromListDTO,
} from "./listCreate.images";

export const LIST_IMAGE_UPLOAD_FAILED_MESSAGE =
  "画像アップロードに失敗しました。後から追加できます。";

/**
 * ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 *
 * 方針:
 * - GET /inventory/list-create/{inventoryId} の response を唯一の正とする。
 * - frontend では model variations API を呼ばない。
 * - priceRows は backend 側で productCategory / model kind に応じた完成形になっている前提。
 *
 * category ごとの表示:
 * - apparel: priceRows[].modelNumber / size / color / rgb
 * - alcohol: priceRows[].modelNumber / volumeValue / volumeUnit
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);

  const raw = await getListCreateRaw(input);
  return mapListCreateDTO(raw);
}

/**
 * list 作成（POST /lists） + 画像（Policy B）
 *
 * Policy B:
 * 1. 画像なしで List を先に作成する
 * 2. 作成済み listId を使って Firebase Storage へ upload する
 * 3. backend に /lists/{listId}/images として image record を作成する
 * 4. primary image を設定する
 *
 * 重要:
 * - List 作成後に画像 upload / image record 登録 / primary image 設定が失敗しても、
 *   List 作成自体は成功として返す。
 * - UI には onImageUploadFailed で
 *   「画像アップロードに失敗しました。後から追加できます。」
 *   を表示する。
 */
export async function createListWithImages(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;

  images?: File[];
  mainImageIndex?: number;

  onImageUploadFailed?: (message: string, error: unknown) => void;
}): Promise<ListDTO> {
  const images = Array.isArray(args.images) ? args.images : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  // 1) build + validate
  const input: CreateListInput = buildCreateListInput({
    params: args.params, // inventoryId(pb__tb) を保持
    listingTitle: args.listingTitle,
    description: args.description,
    priceRows: args.priceRows,
    decision: args.decision,
    assigneeId: args.assigneeId,
  });

  validateCreateListInput(input);

  // 2) create list
  const created = await createListHTTP(input);

  const listId = _internal_getListIdFromListDTO(created);

  if (!listId) {
    throw new Error("created_list_missing_id");
  }

  // 3) images (Policy B)
  //
  // List 作成後の画像失敗は List 作成を失敗扱いにしない。
  // 画面側で「画像アップロードに失敗しました。後から追加できます。」を出せるようにする。
  if (images.length > 0) {
    try {
      await uploadListImagesPolicyB({
        listId,
        files: images,
        mainImageIndex,
        createdBy: auth.currentUser?.uid,
      });
    } catch (error) {
      args.onImageUploadFailed?.(LIST_IMAGE_UPLOAD_FAILED_MESSAGE, error);
    }
  }

  return created;
}
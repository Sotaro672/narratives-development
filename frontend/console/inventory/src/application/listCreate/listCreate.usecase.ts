// frontend/console/inventory/src/application/listCreate/listCreate.usecase.ts

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";
import { getListCreateRaw } from "../../infrastructure/api/listCreateApi";

// ✅ Firebase Auth（uid 取得）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ list create (POST /lists)
import {
  createListHTTP,
  type CreateListInput,
  type ListDTO,
} from "../../../../list/src/infrastructure/http/listRepositoryHTTP";

import type { ResolvedListCreateParams } from "./listCreate.types";
import { buildListCreateFetchInput } from "./listCreate.routing";
import { buildCreateListInput, validateCreateListInput } from "./listCreate.input";
import { s, normalizeListId } from "./listCreate.utils";
import { uploadListImagesPolicyA, _internal_getListIdFromListDTO } from "./listCreate.images";

/**
 * ✅ ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 * - API から raw を取得し、ListCreateDTO として扱う（ListCreateDTO のみを正）
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);

  // getListCreateRaw は any を返すが、この usecase では ListCreateDTO のみを正として扱う
  const raw = await getListCreateRaw(input);
  return raw as ListCreateDTO;
}

/**
 * ✅ list 作成（POST /lists） + 画像（Policy A）
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
}): Promise<ListDTO> {
  const images = Array.isArray(args.images) ? args.images : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  // 1) build + validate
  const input: CreateListInput = buildCreateListInput({
    params: args.params, // ✅ inventoryId(pb__tb) を保持
    listingTitle: args.listingTitle,
    description: args.description,
    priceRows: args.priceRows,
    decision: args.decision,
    assigneeId: args.assigneeId,
  });

  validateCreateListInput(input);

  // 2) create list
  const created = await createListHTTP(input);

  const listIdRaw = _internal_getListIdFromListDTO(
    created,
    s((input as any)?.id) || s((input as any)?.inventoryId),
  );
  const listId = normalizeListId(listIdRaw);

  if (!listId) {
    throw new Error("created_list_missing_id");
  }

  // 3) images (Policy A)
  if (images.length > 0) {
    await uploadListImagesPolicyA({
      listId,
      files: images,
      mainImageIndex,
      createdBy: s(auth.currentUser?.uid) || undefined,
    });
  }

  return created;
}

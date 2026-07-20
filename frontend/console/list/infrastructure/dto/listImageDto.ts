// frontend/console/list/src/infrastructure/dto/listImageDto.ts
export type SaveListImageFromFirebaseStorageInput = {
  listId: string;
  id: string;
  url: string;
  displayOrder: number;
  createdBy?: string;
  createdAt?: string;
};
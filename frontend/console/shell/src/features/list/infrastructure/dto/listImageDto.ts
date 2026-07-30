// frontend/console/shell/src/features/list/infrastructure/dto/listImageDto.ts

export type SaveListImageFromFirebaseStorageInput = {
  listId: string;
  id: string;
  url: string;
  displayOrder: number;
};
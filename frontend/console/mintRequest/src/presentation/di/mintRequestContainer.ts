// frontend/console/mintRequest/src/presentation/di/mintRequestContainer.ts

import type { MintRequestRepository } from "../../application/port/MintRequestRepository";
import { HttpMintRequestRepository } from "../../infrastructure/repository/HttpMintRequestRepository";

export function mintRequestContainer(): {
  mintRequestRepo: MintRequestRepository;
} {
  return {
    mintRequestRepo: new HttpMintRequestRepository(),
  };
}
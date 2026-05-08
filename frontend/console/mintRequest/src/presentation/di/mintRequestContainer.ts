//frontend\console\mintRequest\src\presentation\di\mintRequestContainer.ts
import { HttpMintRequestRepository } from "../../infrastructure/repository/HttpMintRequestRepository";

export function mintRequestContainer() {
  return {
    mintRequestRepo: new HttpMintRequestRepository(),
  };
}

// frontend/inspector/lib/services/product_api.dart
import '../models/inspector_inspection_batch.dart';
import '../models/inspector_product_detail.dart';

import 'product_api/api_client.dart';
import 'product_api/complete_inspection_api.dart';
import 'product_api/fetch_inspection_batch_api.dart';
import 'product_api/fetch_inspector_detail_api.dart';
import 'product_api/submit_inspection_api.dart';
import 'product_api/update_inspection_batch_api.dart';

class ProductApi {
  static final ApiClient _client = ApiClient();

  static Future<InspectorProductDetail> fetchInspectorDetail(String productId) {
    return FetchInspectorDetailApi(_client).fetchInspectorDetail(productId);
  }

  static Future<InspectorInspectionBatch> fetchInspectionBatch(
    String productionId,
  ) {
    return FetchInspectionBatchApi(_client).fetchInspectionBatch(productionId);
  }

  static Future<void> submitInspection({
    required String productId,
    required String result,
  }) {
    return SubmitInspectionApi(
      _client,
    ).submitInspection(productId: productId, result: result);
  }

  static Future<void> updateInspectionBatch({
    required String productionId,
    required String productId,
    required String inspectionResult,
  }) {
    return UpdateInspectionBatchApi(_client).updateInspectionBatch(
      productionId: productionId,
      productId: productId,
      inspectionResult: inspectionResult,
    );
  }

  static Future<void> completeInspection({required String productionId}) {
    return CompleteInspectionApi(
      _client,
    ).completeInspection(productionId: productionId);
  }
}

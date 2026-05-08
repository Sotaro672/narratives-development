// frontend/inspector/lib/services/product_api/update_inspection_batch_api.dart
import 'package:firebase_auth/firebase_auth.dart';

import 'api_client.dart';

class UpdateInspectionBatchApi {
  final ApiClient _client;
  UpdateInspectionBatchApi(this._client);

  Future<void> updateInspectionBatch({
    required String productionId,
    required String productId,
    required String inspectionResult,
  }) async {
    final now = DateTime.now().toUtc().toIso8601String();

    final user = FirebaseAuth.instance.currentUser;
    final inspectedBy = user?.email ?? user?.uid ?? 'unknown';

    final bodyMap = {
      'productionId': productionId,
      'productId': productId,
      'inspectionResult': inspectionResult,
      'inspectedBy': inspectedBy,
      'inspectedAt': now,
    };

    final resp = await _client.patch('/products/inspections', body: bodyMap);

    if (resp.statusCode != 200) {
      throw Exception('inspections 更新に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }
}

// frontend/inspector/lib/services/product_api/fetch_inspection_batch_api.dart
import 'dart:convert';

import '../../models/inspector_inspection_batch.dart';
import 'api_client.dart';

class FetchInspectionBatchApi {
  final ApiClient _client;
  FetchInspectionBatchApi(this._client);

  Future<InspectorInspectionBatch> fetchInspectionBatch(
    String productionId,
  ) async {
    final resp = await _client.get(
      '/products/inspections',
      query: {'productionId': productionId},
    );

    if (resp.statusCode != 200) {
      throw Exception('inspections 取得に失敗しました: ${resp.statusCode} ${resp.body}');
    }

    final body = json.decode(resp.body) as Map<String, dynamic>;
    return InspectorInspectionBatch.fromJson(body);
  }
}

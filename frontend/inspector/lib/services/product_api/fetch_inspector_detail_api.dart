// frontend/inspector/lib/services/product_api/fetch_inspector_detail_api.dart
import 'dart:convert';

import '../../models/inspector_product_detail.dart';
import '../../models/inspector_inspection_batch.dart';
import 'api_client.dart';
import 'fetch_inspection_batch_api.dart';

class FetchInspectorDetailApi {
  final ApiClient _client;
  FetchInspectorDetailApi(this._client);

  Future<InspectorProductDetail> fetchInspectorDetail(String productId) async {
    // 1) detail
    final detailResp = await _client.get('/inspector/products/$productId');
    if (detailResp.statusCode != 200) {
      throw Exception(
        '検品詳細の取得に失敗しました: ${detailResp.statusCode} ${detailResp.body}',
      );
    }

    final detailBody = json.decode(detailResp.body) as Map<String, dynamic>;
    final baseDetail = InspectorProductDetail.fromJson(detailBody);

    // 2) batch（失敗しても detail は返す）
    InspectorInspectionBatch? batch;
    try {
      batch = await FetchInspectionBatchApi(
        _client,
      ).fetchInspectionBatch(baseDetail.productionId);
    } catch (_) {
      batch = null;
    }

    if (batch == null) {
      return baseDetail;
    }

    // 3) current result
    final recordsForThisProduct = batch.inspections
        .where((r) => r.productId == baseDetail.productId)
        .toList();

    final currentResult = recordsForThisProduct.isNotEmpty
        ? (recordsForThisProduct.first.inspectionResult ?? '')
        : baseDetail.inspectionResult;

    // 4) merge
    return InspectorProductDetail(
      productId: baseDetail.productId,
      productionId: baseDetail.productionId,
      modelId: baseDetail.modelId,
      productBlueprintId: baseDetail.productBlueprintId,
      modelNumber: baseDetail.modelNumber,
      size: baseDetail.size,
      measurements: baseDetail.measurements,
      color: baseDetail.color,
      productBlueprint: baseDetail.productBlueprint,
      inspections: batch.inspections,
      inspectionResult: currentResult,
    );
  }
}

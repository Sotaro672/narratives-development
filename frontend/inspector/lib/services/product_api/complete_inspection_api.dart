// frontend/inspector/lib/services/product_api/complete_inspection_api.dart
import 'api_client.dart';

class CompleteInspectionApi {
  final ApiClient _client;
  CompleteInspectionApi(this._client);

  Future<void> completeInspection({required String productionId}) async {
    final bodyMap = {'productionId': productionId};

    final resp = await _client.patch(
      '/products/inspections/complete',
      body: bodyMap,
    );

    if (resp.statusCode != 200) {
      throw Exception('検品完了処理に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }
}

// frontend/inspector/lib/services/product_api/submit_inspection_api.dart
import 'package:firebase_auth/firebase_auth.dart';

import 'api_client.dart';

class SubmitInspectionApi {
  final ApiClient _client;
  SubmitInspectionApi(this._client);

  Future<void> submitInspection({
    required String productId,
    required String result,
  }) async {
    final now = DateTime.now().toUtc().toIso8601String();

    final user = FirebaseAuth.instance.currentUser;
    final inspectedBy = user?.email ?? user?.uid ?? 'unknown';

    final bodyMap = {
      'inspectionResult': result == 'passed' ? 'passed' : 'failed',
      'inspectedAt': now,
      'inspectedBy': inspectedBy,
    };

    final resp = await _client.patch('/products/$productId', body: bodyMap);

    if (resp.statusCode != 200) {
      throw Exception('検品結果の送信に失敗しました: ${resp.statusCode} ${resp.body}');
    }
  }
}

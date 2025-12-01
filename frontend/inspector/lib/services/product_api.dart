import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

/// Cloud Run backend のベースURL
const String kBackendBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

class ProductApi {
  /// 検品結果を送信
  ///
  /// [result] は 'pass' / 'fail' / 'notYet' など domain 側の InspectionResult に合わせて使用
  static Future<void> submitInspection({
    required String productId,
    required String result,
  }) async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) {
      throw Exception('未ログインです');
    }

    final idToken = await user.getIdToken();
    final uri = Uri.parse('$kBackendBaseUrl/products/$productId');

    final body = <String, dynamic>{
      'inspectionResult': result, // e.g. "pass" or "fail"
      'connectedToken': null, // 今は未使用
      'inspectedAt': DateTime.now().toUtc().toIso8601String(),
      'inspectedBy': user.uid,
    };

    final res = await http.patch(
      uri,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $idToken',
      },
      body: jsonEncode(body),
    );

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw Exception(
        'PATCH /products/$productId failed: '
        '${res.statusCode} ${res.reasonPhrase} ${res.body}',
      );
    }
  }
}

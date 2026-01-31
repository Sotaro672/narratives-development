// frontend/inspector/lib/services/inspection_api.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../config/backend_config.dart';

/// 検品結果をバックエンドに送信する
Future<void> patchInspection({
  required String productionId,
  required String productId,
  required String inspectionResult, // 'passed' / 'failed' / 'notYet' など
  required DateTime inspectedAt,
  String? status, // 'completed' など必要なら
}) async {
  final user = FirebaseAuth.instance.currentUser;
  if (user == null) {
    throw Exception('ログインしていません');
  }

  // 🔑 Firebase ID トークン取得
  final idToken = await user.getIdToken();

  final uri = Uri.parse('$backendBaseUrl/products/inspections');
  final resp = await http.patch(
    uri,
    headers: {
      'Authorization': 'Bearer $idToken', // ★ AuthMiddleware に渡る
      'Content-Type': 'application/json',
    },
    body: jsonEncode({
      'productionId': productionId,
      'productId': productId,
      'inspectionResult': inspectionResult,
      // inspectedBy はサーバ側で決定する方針
      'inspectedAt': inspectedAt.toUtc().toIso8601String(),
      if (status != null) 'status': status,
    }),
  );

  if (resp.statusCode != 200) {
    throw Exception('検品更新に失敗しました: ${resp.statusCode} ${resp.body}');
  }
}

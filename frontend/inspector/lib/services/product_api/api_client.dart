// frontend/inspector/lib/services/product_api/api_client.dart
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

class ApiClient {
  static const String baseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  Future<String> getIdToken() async {
    final user = FirebaseAuth.instance.currentUser;
    final token = await user?.getIdToken();
    if (token == null || token.isEmpty) {
      throw Exception('ログイン情報が見つかりません（IDトークンが取得できませんでした）');
    }
    return token;
  }

  Future<http.Response> get(String path, {Map<String, String>? query}) async {
    final token = await getIdToken();
    final uri = Uri.parse('$baseUrl$path').replace(queryParameters: query);
    return http.get(uri, headers: {'Authorization': 'Bearer $token'});
  }

  Future<http.Response> patch(
    String path, {
    Map<String, String>? query,
    required Map<String, dynamic> body,
  }) async {
    final token = await getIdToken();
    final uri = Uri.parse('$baseUrl$path').replace(queryParameters: query);
    return http.patch(
      uri,
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: json.encode(body),
    );
  }
}

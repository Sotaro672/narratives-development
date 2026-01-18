// frontend/mall/lib/features/preview/infrastructure/scan_transfer_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';
import 'http_common.dart';
import 'models.dart';

class ScanTransferRepositoryHttp {
  ScanTransferRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  String _pickAuth(Map<String, String> h) {
    // absorb "Authorization" vs "authorization"
    final a1 = (h['Authorization'] ?? '').trim();
    if (a1.isNotEmpty) return a1;
    final a2 = (h['authorization'] ?? '').trim();
    if (a2.isNotEmpty) return a2;
    return '';
  }

  /// POST /mall/me/orders/scan/transfer
  ///
  /// NEW CONTRACT:
  /// - Authorization header is REQUIRED
  /// - Body: { productId } only
  Future<MallScanTransferResponse> transferScanPurchased({
    required String productId,
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final pid = productId.trim();
    if (pid.isEmpty) {
      throw ArgumentError('productId is empty');
    }

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse('$b/mall/me/orders/scan/transfer');

    // Base JSON headers + caller headers
    final mergedHeaders = <String, String>{...jsonPostHeaders()};
    if (headers != null) {
      mergedHeaders.addAll(headers);
    }

    // Ensure Authorization exists and normalize key to "Authorization"
    final auth = _pickAuth(mergedHeaders);
    if (auth.isEmpty) {
      throw ArgumentError('Authorization header is required for transfer');
    }
    mergedHeaders['Authorization'] = auth;

    // ✅ body must be productId only
    final body = jsonEncode({'productId': pid});

    final res = await _client.post(uri, headers: mergedHeaders, body: body);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'transferScanPurchased failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    // absorb both shapes:
    // 1) { "data": { ... } }
    // 2) { ... }
    final root = decoded.cast<String, dynamic>();
    final data = root['data'];
    if (data is Map) {
      return MallScanTransferResponse.fromJson(data.cast<String, dynamic>());
    }
    return MallScanTransferResponse.fromJson(root);
  }

  @Deprecated(
    'avatarId is resolved on server; use transferScanPurchased(productId: ...) instead.',
  )
  Future<MallScanTransferResponse> transferScanPurchasedByAvatarId({
    required String avatarId,
    required String productId,
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    return transferScanPurchased(
      productId: productId,
      baseUrl: baseUrl,
      headers: headers,
    );
  }
}

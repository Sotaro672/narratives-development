// frontend/mall/lib/features/preview/infrastructure/scan_verify_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';
import 'http_common.dart';
import 'models.dart';

class ScanVerifyRepositoryHttp {
  ScanVerifyRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  Future<MallScanVerifyResponse> verifyScanPurchasedByAvatarId({
    required String avatarId,
    required String productId,
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final aid = avatarId.trim();
    final pid = productId.trim();

    if (aid.isEmpty) {
      throw ArgumentError('avatarId is empty');
    }
    if (pid.isEmpty) {
      throw ArgumentError('productId is empty');
    }

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse('$b/mall/me/orders/scan/verify');

    final mergedHeaders = <String, String>{...jsonPostHeaders()};
    if (headers != null) {
      mergedHeaders.addAll(headers);
    }

    final body = jsonEncode({'avatarId': aid, 'productId': pid});

    final res = await _client.post(uri, headers: mergedHeaders, body: body);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'verifyScanPurchasedByAvatarId failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    return MallScanVerifyResponse.fromJson(decoded.cast<String, dynamic>());
  }
}

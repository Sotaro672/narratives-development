// frontend/mall/lib/features/preview/infrastructure/preview_repository_http.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';
import 'http_common.dart';
import 'models.dart';

class PreviewRepositoryHttp {
  PreviewRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  /// ✅ Public preview (QR entry)
  /// GET /mall/preview?productId=...
  ///
  /// NOTE:
  /// - owner resolve はバックエンド側（preview handler）で解決して返す方針。
  /// - フロントから /mall/owner-resolve を追加で叩かない（404回避）。
  Future<MallPreviewResponse> fetchPreviewByProductId(
    String productId, {
    String? baseUrl,
  }) async {
    final id = productId.trim();
    if (id.isEmpty) {
      throw ArgumentError('productId is empty');
    }

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/preview',
    ).replace(queryParameters: {'productId': id});

    final res = await _client.get(uri, headers: jsonHeaders());

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchPreviewByProductId failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    // ✅ owner はバックエンドが返すものをそのまま使う
    return MallPreviewResponse.fromJson(decoded.cast<String, dynamic>());
  }

  /// ✅ Authenticated preview (user scope)
  /// GET /mall/me/preview?productId=...
  ///
  /// NOTE:
  /// - owner resolve はバックエンド側（preview_me handler）で解決して返す方針。
  /// - フロントから /mall/me/owner-resolve を追加で叩かない（404回避）。
  Future<MallPreviewResponse> fetchMyPreviewByProductId(
    String productId, {
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final id = productId.trim();
    if (id.isEmpty) {
      throw ArgumentError('productId is empty');
    }

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/me/preview',
    ).replace(queryParameters: {'productId': id});

    final mergedHeaders = <String, String>{...jsonHeaders()};
    if (headers != null) {
      mergedHeaders.addAll(headers);
    }

    final res = await _client.get(uri, headers: mergedHeaders);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchMyPreviewByProductId failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    // ✅ owner はバックエンドが返すものをそのまま使う
    return MallPreviewResponse.fromJson(decoded.cast<String, dynamic>());
  }
}

// frontend/mall/lib/features/preview/infrastructure/repository.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジック（single source of truth）
import '../../../app/config/api_base.dart';

/// Preview response (minimal DTO).
///
/// backend (想定):
/// - GET /mall/preview?productId=...         (public / QR entry)
/// - GET /mall/me/preview?productId=...      (authenticated scope)
///
/// まずは productId と modelId だけ取得できればOK。
class MallPreviewResponse {
  MallPreviewResponse({required this.productId, required this.modelId});

  /// Product ID（= QR の {productId}）
  final String productId;

  /// Model ID（= 製造モデル）
  final String modelId;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  /// wrapper/ネスト揺れを best-effort で吸収して DTO 化
  factory MallPreviewResponse.fromJson(Map<String, dynamic> j) {
    // wrapper 吸収: {data:{...}} を許容
    final maybeData = j['data'];
    if (maybeData is Map<String, dynamic>) {
      return MallPreviewResponse.fromJson(maybeData);
    }
    if (maybeData is Map) {
      return MallPreviewResponse.fromJson(Map<String, dynamic>.from(maybeData));
    }

    // product がネストして返るケース（previewがproductを内包する設計になりがち）
    final prod = j['product'];
    final Map<String, dynamic>? pm = (prod is Map<String, dynamic>)
        ? prod
        : (prod is Map ? Map<String, dynamic>.from(prod) : null);

    // ---- productId ----
    final productIdRaw = _s(j['productId']);
    final idRaw = _s(j['id']); // 返却が product そのものだった場合に備える
    final nestedProductIdRaw = pm != null
        ? (_s(pm['id']).isNotEmpty ? _s(pm['id']) : _s(pm['productId']))
        : '';
    final productId = productIdRaw.isNotEmpty
        ? productIdRaw
        : (nestedProductIdRaw.isNotEmpty ? nestedProductIdRaw : idRaw);

    // ---- modelId ----
    // 想定: 直下に modelId または product.modelId
    var modelIdRaw = _s(j['modelId']);
    if (modelIdRaw.isEmpty && pm != null) {
      modelIdRaw = _s(pm['modelId']);
    }
    if (modelIdRaw.isEmpty) {
      modelIdRaw = _s(j['model_id']); // 念のため
    }

    return MallPreviewResponse(productId: productId, modelId: modelIdRaw);
  }
}

class PreviewRepositoryHttp {
  PreviewRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  static String _normalizeBaseUrl(String s) {
    var v = s.trim();
    if (v.isEmpty) return v;
    while (v.endsWith('/')) {
      v = v.substring(0, v.length - 1);
    }
    return v;
  }

  Map<String, String> _jsonHeaders() => const {'Accept': 'application/json'};

  /// ✅ Public preview (QR entry)
  /// GET /mall/preview?productId=...
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
    final b = _normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/preview',
    ).replace(queryParameters: {'productId': id});

    final res = await _client.get(uri, headers: _jsonHeaders());

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchPreviewByProductId failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is Map) {
      return MallPreviewResponse.fromJson(decoded.cast<String, dynamic>());
    }
    throw const FormatException('invalid json shape (expected object)');
  }

  /// ✅ Authenticated preview (user scope)
  /// GET /mall/me/preview?productId=...
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
    final b = _normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/me/preview',
    ).replace(queryParameters: {'productId': id});

    final mergedHeaders = <String, String>{
      ..._jsonHeaders(),
      if (headers != null) ...headers,
    };

    final res = await _client.get(uri, headers: mergedHeaders);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchMyPreviewByProductId failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is Map) {
      return MallPreviewResponse.fromJson(decoded.cast<String, dynamic>());
    }
    throw const FormatException('invalid json shape (expected object)');
  }
}

class HttpException implements Exception {
  HttpException(this.message, {this.url, this.body});

  final String message;
  final String? url;
  final String? body;

  @override
  String toString() {
    final u = url == null ? '' : ' url=$url';
    final b = body == null
        ? ''
        : ' body=${body!.length > 300 ? body!.substring(0, 300) : body}';
    return 'HttpException($message$u$b)';
  }
}

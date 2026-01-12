// frontend/mall/lib/features/preview/infrastructure/repository.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジック（single source of truth）
import '../../../app/config/api_base.dart';

/// Preview response DTO.
///
/// backend (想定):
/// - GET /mall/preview?productId=...         (public / QR entry)
/// - GET /mall/me/preview?productId=...      (authenticated scope)
///
/// まずは productId と modelId + 表示用メタ（型番/サイズ/色/RGB）を取得できればOK。
class MallPreviewResponse {
  MallPreviewResponse({
    required this.productId,
    required this.modelId,
    this.modelNumber = '',
    this.size = '',
    this.color = '',
    this.rgb = 0,
    this.measurements,
    this.productBlueprintPatch,
    this.token, // ✅ NEW
  });

  /// Product ID（= QR の {productId}）
  final String productId;

  /// Model ID（= 製造モデル / variationId 想定）
  final String modelId;

  /// 型番（例: "ABC-123"）
  final String modelNumber;

  /// サイズ（例: "M"）
  final String size;

  /// 色名（例: "Navy"）
  final String color;

  /// RGB（0xRRGGBB を想定した int）
  final int rgb;

  /// 任意: 計測値（採寸）
  final Map<String, int>? measurements;

  /// ✅ NEW: productBlueprintPatch 全体（JSONそのまま）
  /// - backend の data.productBlueprintPatch をそのまま保持する
  final Map<String, dynamic>? productBlueprintPatch;

  /// ✅ NEW: tokens/{productId}（あれば）
  final MallTokenInfo? token;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static int _i(dynamic v) {
    if (v == null) return 0;
    if (v is int) return v;
    if (v is num) return v.toInt();
    final s = v.toString().trim();
    if (s.isEmpty) return 0;

    // "0xRRGGBB" も許容（念のため）
    if (s.startsWith('0x') || s.startsWith('0X')) {
      return int.tryParse(s.substring(2), radix: 16) ?? 0;
    }
    return int.tryParse(s) ?? 0;
  }

  static Map<String, int>? _measurements(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, int>) return v;

    if (v is Map) {
      final out = <String, int>{};
      v.forEach((k, val) {
        final key = _s(k);
        if (key.isEmpty) return;
        out[key] = _i(val);
      });
      return out.isEmpty ? null : out;
    }
    return null;
  }

  static Map<String, dynamic>? _jsonObject(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return Map<String, dynamic>.from(v);
    return null;
  }

  static MallTokenInfo? _token(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, dynamic>) return MallTokenInfo.fromJson(v);
    if (v is Map) return MallTokenInfo.fromJson(Map<String, dynamic>.from(v));
    return null;
  }

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

    // ---- meta: modelNumber / size / color / rgb ----
    var modelNumberRaw = _s(j['modelNumber']);
    if (modelNumberRaw.isEmpty && pm != null) {
      modelNumberRaw = _s(pm['modelNumber']);
    }

    var sizeRaw = _s(j['size']);
    if (sizeRaw.isEmpty && pm != null) {
      sizeRaw = _s(pm['size']);
    }

    var colorRaw = _s(j['color']);
    if (colorRaw.isEmpty && pm != null) {
      colorRaw = _s(pm['color']);
    }

    var rgbRaw = _i(j['rgb']);
    if (rgbRaw == 0 && pm != null) {
      rgbRaw = _i(pm['rgb']);
    }

    // ✅ measurements は「直下」or「product内」どちらでも拾う
    final measurementsRaw =
        _measurements(j['measurements']) ??
        (pm != null ? _measurements(pm['measurements']) : null);

    // ✅ productBlueprintPatch は「直下」or「product内」どちらでも拾う
    final productBlueprintPatchRaw =
        _jsonObject(j['productBlueprintPatch']) ??
        (pm != null ? _jsonObject(pm['productBlueprintPatch']) : null);

    // ✅ token は「直下 token」or「product内 token」どちらでも拾う
    final tokenRaw =
        _token(j['token']) ?? (pm != null ? _token(pm['token']) : null);

    return MallPreviewResponse(
      productId: productId,
      modelId: modelIdRaw,
      modelNumber: modelNumberRaw,
      size: sizeRaw,
      color: colorRaw,
      rgb: rgbRaw,
      measurements: measurementsRaw,
      productBlueprintPatch: productBlueprintPatchRaw,
      token: tokenRaw,
    );
  }
}

/// ✅ NEW: tokens/{productId} の最小ビュー
class MallTokenInfo {
  MallTokenInfo({
    required this.productId,
    this.brandId = '',
    this.tokenBlueprintId = '',
    this.mintAddress = '',
    this.onChainTxSignature = '',
  });

  final String productId;
  final String brandId;
  final String tokenBlueprintId;
  final String mintAddress;
  final String onChainTxSignature;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  factory MallTokenInfo.fromJson(Map<String, dynamic> j) {
    // wrapper 吸収: {data:{...}} が混ざっても耐える
    final maybeData = j['data'];
    if (maybeData is Map<String, dynamic>) {
      return MallTokenInfo.fromJson(maybeData);
    }
    if (maybeData is Map) {
      return MallTokenInfo.fromJson(Map<String, dynamic>.from(maybeData));
    }

    return MallTokenInfo(
      productId: _s(j['productId']),
      brandId: _s(j['brandId']),
      tokenBlueprintId: _s(j['tokenBlueprintId']),
      mintAddress: _s(j['mintAddress']),
      onChainTxSignature: _s(j['onChainTxSignature']),
    );
  }

  Map<String, dynamic> toJson() => {
    'productId': productId,
    'brandId': brandId,
    'tokenBlueprintId': tokenBlueprintId,
    'mintAddress': mintAddress,
    'onChainTxSignature': onChainTxSignature,
  };
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

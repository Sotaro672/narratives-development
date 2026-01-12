// frontend/mall/lib/features/preview/infrastructure/repository.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジック（single source of truth）
import '../../../app/config/api_base.dart';

/// ✅ owner-resolve response DTO.
///
/// backend (想定):
/// - GET /mall/owner-resolve?walletAddress=...     (public)
/// - GET /mall/me/owner-resolve?walletAddress=...  (auth)
///
/// 返却例（どちらでも許容）:
/// - { "brandId": "...", "avatarId": "..." }
/// - { "data": { "brandId": "...", "avatarId": "..." } }
class MallOwnerInfo {
  MallOwnerInfo({this.brandId = '', this.avatarId = ''});

  final String brandId;
  final String avatarId;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  factory MallOwnerInfo.fromJson(Map<String, dynamic> j) {
    // wrapper 吸収: {data:{...}} を許容
    final maybeData = j['data'];
    if (maybeData is Map<String, dynamic>) {
      return MallOwnerInfo.fromJson(maybeData);
    }
    if (maybeData is Map) {
      return MallOwnerInfo.fromJson(Map<String, dynamic>.from(maybeData));
    }

    return MallOwnerInfo(
      brandId: _s(j['brandId']),
      avatarId: _s(j['avatarId']),
    );
  }

  Map<String, dynamic> toJson() => {'brandId': brandId, 'avatarId': avatarId};
}

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
    this.token, // ✅ token info (tokens/{productId})
    this.owner, // ✅ owner resolve result (walletAddress -> brandId/avatarId)
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

  /// productBlueprintPatch 全体（JSONそのまま）
  /// - backend の data.productBlueprintPatch をそのまま保持する
  final Map<String, dynamic>? productBlueprintPatch;

  /// ✅ tokens/{productId}（あれば）
  final MallTokenInfo? token;

  /// ✅ owner resolve result（あれば）
  final MallOwnerInfo? owner;

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

  static MallOwnerInfo? _owner(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, dynamic>) return MallOwnerInfo.fromJson(v);
    if (v is Map) return MallOwnerInfo.fromJson(Map<String, dynamic>.from(v));
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

    // ✅ owner は「直下 owner」or「product内 owner」どちらでも拾う
    final ownerRaw =
        _owner(j['owner']) ?? (pm != null ? _owner(pm['owner']) : null);

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
      owner: ownerRaw,
    );
  }
}

/// ✅ tokens/{productId} の最小ビュー（backend TokenInfo に追随）
///
/// 方針:
/// - docID=productId（既存）
/// - tokens には mintAddress / onChainTxSignature / mintedAt / brandId を保存
/// - 体感速度向上のため、toAddress / metadataUri を tokens にキャッシュして即表示に使う
/// - productId / tokenBlueprintId は Firestore に保存しない（productId は docID で十分）
class MallTokenInfo {
  MallTokenInfo({
    required this.productId,
    this.brandId = '',
    this.toAddress = '',
    this.metadataUri = '',
    this.mintAddress = '',
    this.onChainTxSignature = '',
    this.mintedAt = '',
  });

  /// docID (=productId) をレスポンスに含める用途
  final String productId;

  final String brandId;

  /// ✅ Off-chain cache (for faster UI)
  final String toAddress;
  final String metadataUri;

  /// On-chain results
  final String mintAddress;
  final String onChainTxSignature;

  /// mintedAt（RFC3339 など、backend側の整形に依存）
  final String mintedAt;

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
      toAddress: _s(j['toAddress']),
      metadataUri: _s(j['metadataUri']),
      mintAddress: _s(j['mintAddress']),
      onChainTxSignature: _s(j['onChainTxSignature']),
      mintedAt: _s(j['mintedAt']),
    );
  }

  Map<String, dynamic> toJson() => {
    'productId': productId,
    'brandId': brandId,
    'toAddress': toAddress,
    'metadataUri': metadataUri,
    'mintAddress': mintAddress,
    'onChainTxSignature': onChainTxSignature,
    'mintedAt': mintedAt,
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

  /// ✅ Public owner resolve
  /// GET /mall/owner-resolve?walletAddress=...
  Future<MallOwnerInfo?> fetchOwnerResolvePublic(
    String walletAddress, {
    String? baseUrl,
  }) async {
    final addr = walletAddress.trim();
    if (addr.isEmpty) return null;

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = _normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/owner-resolve',
    ).replace(queryParameters: {'walletAddress': addr});

    final res = await _client.get(uri, headers: _jsonHeaders());

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchOwnerResolvePublic failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is Map) {
      return MallOwnerInfo.fromJson(decoded.cast<String, dynamic>());
    }
    throw const FormatException('invalid json shape (expected object)');
  }

  /// ✅ Authenticated owner resolve
  /// GET /mall/me/owner-resolve?walletAddress=...
  Future<MallOwnerInfo?> fetchOwnerResolveMe(
    String walletAddress, {
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final addr = walletAddress.trim();
    if (addr.isEmpty) return null;

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = _normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/me/owner-resolve',
    ).replace(queryParameters: {'walletAddress': addr});

    final mergedHeaders = <String, String>{
      ..._jsonHeaders(),
      if (headers != null) ...headers,
    };

    final res = await _client.get(uri, headers: mergedHeaders);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchOwnerResolveMe failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is Map) {
      return MallOwnerInfo.fromJson(decoded.cast<String, dynamic>());
    }
    throw const FormatException('invalid json shape (expected object)');
  }

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
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    final preview = MallPreviewResponse.fromJson(
      decoded.cast<String, dynamic>(),
    );

    // ✅ owner resolve を追加で叩く（best-effort）
    // public では token.toAddress が取れている場合に /mall/owner-resolve を呼ぶ
    final toAddr = (preview.token?.toAddress ?? '').trim();
    if (toAddr.isNotEmpty) {
      try {
        final owner = await fetchOwnerResolvePublic(toAddr, baseUrl: b);
        return MallPreviewResponse(
          productId: preview.productId,
          modelId: preview.modelId,
          modelNumber: preview.modelNumber,
          size: preview.size,
          color: preview.color,
          rgb: preview.rgb,
          measurements: preview.measurements,
          productBlueprintPatch: preview.productBlueprintPatch,
          token: preview.token,
          owner: owner,
        );
      } catch (_) {
        // best-effort: owner が取れなくても preview は返す
        return preview;
      }
    }

    return preview;
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
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    final preview = MallPreviewResponse.fromJson(
      decoded.cast<String, dynamic>(),
    );

    // ✅ owner resolve を追加で叩く（best-effort）
    // me では token.toAddress が取れている場合に /mall/me/owner-resolve を呼ぶ
    final toAddr = (preview.token?.toAddress ?? '').trim();
    if (toAddr.isNotEmpty) {
      try {
        final owner = await fetchOwnerResolveMe(
          toAddr,
          baseUrl: b,
          headers: headers,
        );
        return MallPreviewResponse(
          productId: preview.productId,
          modelId: preview.modelId,
          modelNumber: preview.modelNumber,
          size: preview.size,
          color: preview.color,
          rgb: preview.rgb,
          measurements: preview.measurements,
          productBlueprintPatch: preview.productBlueprintPatch,
          token: preview.token,
          owner: owner,
        );
      } catch (_) {
        // best-effort: owner が取れなくても preview は返す
        return preview;
      }
    }

    return preview;
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

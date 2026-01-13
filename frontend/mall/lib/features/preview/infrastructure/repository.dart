// frontend/mall/lib/features/preview/infrastructure/repository.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジック（single source of truth）
import '../../../app/config/api_base.dart';

class MallOwnerInfo {
  MallOwnerInfo({this.brandId = '', this.avatarId = ''});

  final String brandId;
  final String avatarId;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  factory MallOwnerInfo.fromJson(Map<String, dynamic> j) {
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

class MallModelTokenPair {
  MallModelTokenPair({this.modelId = '', this.tokenBlueprintId = ''});

  final String modelId;
  final String tokenBlueprintId;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  factory MallModelTokenPair.fromJson(Map<String, dynamic> j) {
    return MallModelTokenPair(
      modelId: _s(j['modelId']),
      tokenBlueprintId: _s(j['tokenBlueprintId']),
    );
  }

  Map<String, dynamic> toJson() => {
    'modelId': modelId,
    'tokenBlueprintId': tokenBlueprintId,
  };
}

class MallScanVerifyResponse {
  MallScanVerifyResponse({
    required this.avatarId,
    required this.productId,
    required this.scannedModelId,
    required this.scannedTokenBlueprintId,
    required this.purchasedPairs,
    required this.matched,
    this.match,
  });

  final String avatarId;
  final String productId;

  final String scannedModelId;
  final String scannedTokenBlueprintId;

  final List<MallModelTokenPair> purchasedPairs;

  final bool matched;
  final MallModelTokenPair? match;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static bool _b(dynamic v) {
    if (v == null) {
      return false;
    }
    if (v is bool) {
      return v;
    }
    if (v is num) {
      return v != 0;
    }
    final s = v.toString().trim().toLowerCase();
    return s == 'true' || s == '1' || s == 'yes';
  }

  static List<MallModelTokenPair> _pairs(dynamic v) {
    if (v == null) {
      return <MallModelTokenPair>[];
    }
    if (v is! List) {
      return <MallModelTokenPair>[];
    }

    return v
        .map((e) {
          if (e is Map<String, dynamic>) {
            return MallModelTokenPair.fromJson(e);
          }
          if (e is Map) {
            return MallModelTokenPair.fromJson(Map<String, dynamic>.from(e));
          }
          return null;
        })
        .whereType<MallModelTokenPair>()
        .toList();
  }

  static MallModelTokenPair? _pair(dynamic v) {
    if (v == null) {
      return null;
    }
    if (v is Map<String, dynamic>) {
      return MallModelTokenPair.fromJson(v);
    }
    if (v is Map) {
      return MallModelTokenPair.fromJson(Map<String, dynamic>.from(v));
    }
    return null;
  }

  factory MallScanVerifyResponse.fromJson(Map<String, dynamic> j) {
    final maybeData = j['data'];
    if (maybeData is Map<String, dynamic>) {
      return MallScanVerifyResponse.fromJson(maybeData);
    }
    if (maybeData is Map) {
      return MallScanVerifyResponse.fromJson(
        Map<String, dynamic>.from(maybeData),
      );
    }

    return MallScanVerifyResponse(
      avatarId: _s(j['avatarId']),
      productId: _s(j['productId']),
      scannedModelId: _s(j['scannedModelId']),
      scannedTokenBlueprintId: _s(j['scannedTokenBlueprintId']),
      purchasedPairs: _pairs(j['purchasedPairs']),
      matched: _b(j['matched']),
      match: _pair(j['match']),
    );
  }

  Map<String, dynamic> toJson() => {
    'avatarId': avatarId,
    'productId': productId,
    'scannedModelId': scannedModelId,
    'scannedTokenBlueprintId': scannedTokenBlueprintId,
    'purchasedPairs': purchasedPairs.map((e) => e.toJson()).toList(),
    'matched': matched,
    'match': match?.toJson(),
  };
}

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
    this.token,
    this.owner,
  });

  final String productId;
  final String modelId;

  final String modelNumber;
  final String size;
  final String color;
  final int rgb;

  final Map<String, int>? measurements;
  final Map<String, dynamic>? productBlueprintPatch;

  final MallTokenInfo? token;
  final MallOwnerInfo? owner;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static int _i(dynamic v) {
    if (v == null) {
      return 0;
    }
    if (v is int) {
      return v;
    }
    if (v is num) {
      return v.toInt();
    }
    final s = v.toString().trim();
    if (s.isEmpty) {
      return 0;
    }

    if (s.startsWith('0x') || s.startsWith('0X')) {
      return int.tryParse(s.substring(2), radix: 16) ?? 0;
    }
    return int.tryParse(s) ?? 0;
  }

  static Map<String, int>? _measurements(dynamic v) {
    if (v == null) {
      return null;
    }
    if (v is Map<String, int>) {
      return v;
    }

    if (v is Map) {
      final out = <String, int>{};
      v.forEach((k, val) {
        final key = _s(k);
        if (key.isEmpty) {
          return;
        }
        out[key] = _i(val);
      });
      if (out.isEmpty) {
        return null;
      }
      return out;
    }
    return null;
  }

  static Map<String, dynamic>? _jsonObject(dynamic v) {
    if (v == null) {
      return null;
    }
    if (v is Map<String, dynamic>) {
      return v;
    }
    if (v is Map) {
      return Map<String, dynamic>.from(v);
    }
    return null;
  }

  static MallTokenInfo? _token(dynamic v) {
    if (v == null) {
      return null;
    }
    if (v is Map<String, dynamic>) {
      return MallTokenInfo.fromJson(v);
    }
    if (v is Map) {
      return MallTokenInfo.fromJson(Map<String, dynamic>.from(v));
    }
    return null;
  }

  static MallOwnerInfo? _owner(dynamic v) {
    if (v == null) {
      return null;
    }
    if (v is Map<String, dynamic>) {
      return MallOwnerInfo.fromJson(v);
    }
    if (v is Map) {
      return MallOwnerInfo.fromJson(Map<String, dynamic>.from(v));
    }
    return null;
  }

  factory MallPreviewResponse.fromJson(Map<String, dynamic> j) {
    final maybeData = j['data'];
    if (maybeData is Map<String, dynamic>) {
      return MallPreviewResponse.fromJson(maybeData);
    }
    if (maybeData is Map) {
      return MallPreviewResponse.fromJson(Map<String, dynamic>.from(maybeData));
    }

    final prod = j['product'];
    final Map<String, dynamic>? pm;
    if (prod is Map<String, dynamic>) {
      pm = prod;
    } else if (prod is Map) {
      pm = Map<String, dynamic>.from(prod);
    } else {
      pm = null;
    }

    final productIdRaw = _s(j['productId']);
    final idRaw = _s(j['id']);
    final nestedProductIdRaw = pm != null
        ? (_s(pm['id']).isNotEmpty ? _s(pm['id']) : _s(pm['productId']))
        : '';
    final productId = productIdRaw.isNotEmpty
        ? productIdRaw
        : (nestedProductIdRaw.isNotEmpty ? nestedProductIdRaw : idRaw);

    var modelIdRaw = _s(j['modelId']);
    if (modelIdRaw.isEmpty && pm != null) {
      modelIdRaw = _s(pm['modelId']);
    }

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

    final measurementsRaw =
        _measurements(j['measurements']) ??
        (pm != null ? _measurements(pm['measurements']) : null);

    final productBlueprintPatchRaw =
        _jsonObject(j['productBlueprintPatch']) ??
        (pm != null ? _jsonObject(pm['productBlueprintPatch']) : null);

    final tokenRaw =
        _token(j['token']) ?? (pm != null ? _token(pm['token']) : null);

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

  final String productId;

  final String brandId;

  final String toAddress;
  final String metadataUri;

  final String mintAddress;
  final String onChainTxSignature;

  final String mintedAt;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  factory MallTokenInfo.fromJson(Map<String, dynamic> j) {
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
    if (v.isEmpty) {
      return v;
    }
    while (v.endsWith('/')) {
      v = v.substring(0, v.length - 1);
    }
    return v;
  }

  Map<String, String> _jsonHeaders() => const {'Accept': 'application/json'};

  Map<String, String> _jsonPostHeaders() => const {
    'Accept': 'application/json',
    'Content-Type': 'application/json',
  };

  Future<MallOwnerInfo?> fetchOwnerResolvePublic(
    String walletAddress, {
    String? baseUrl,
  }) async {
    final addr = walletAddress.trim();
    if (addr.isEmpty) {
      return null;
    }

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

  Future<MallOwnerInfo?> fetchOwnerResolveMe(
    String walletAddress, {
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final addr = walletAddress.trim();
    if (addr.isEmpty) {
      return null;
    }

    final resolvedBase = (baseUrl ?? '').trim().isNotEmpty
        ? baseUrl!.trim()
        : resolveApiBase();
    final b = _normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse(
      '$b/mall/me/owner-resolve',
    ).replace(queryParameters: {'walletAddress': addr});

    final mergedHeaders = <String, String>{..._jsonHeaders()};
    if (headers != null) {
      mergedHeaders.addAll(headers);
    }

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
        return preview;
      }
    }

    return preview;
  }

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

    final mergedHeaders = <String, String>{..._jsonHeaders()};
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

    final preview = MallPreviewResponse.fromJson(
      decoded.cast<String, dynamic>(),
    );

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
        return preview;
      }
    }

    return preview;
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
    final b = _normalizeBaseUrl(resolvedBase);

    final uri = Uri.parse('$b/mall/me/orders/scan/verify');

    final mergedHeaders = <String, String>{..._jsonPostHeaders()};
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

// frontend/mall/lib/features/preview/infrastructure/models.dart
import 'dart:convert';

class MallOwnerInfo {
  MallOwnerInfo({
    this.brandId = '',
    this.avatarId = '',
    this.brandName = '',
    this.avatarName = '',
  });

  final String brandId;
  final String avatarId;

  // ✅ NEW: resolved display names (best-effort)
  // - ownerType=brand のとき brandName が入る想定
  // - ownerType=avatar のとき avatarName が入る想定
  final String brandName;
  final String avatarName;

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
      brandName: _s(j['brandName']),
      avatarName: _s(j['avatarName']),
    );
  }

  Map<String, dynamic> toJson() => {
    'brandId': brandId,
    'avatarId': avatarId,
    'brandName': brandName,
    'avatarName': avatarName,
  };
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

  String toPrettyJson() {
    try {
      return const JsonEncoder.withIndent('  ').convert(toJson());
    } catch (_) {
      return toString();
    }
  }

  Map<String, dynamic> toJson() => {
    'productId': productId,
    'modelId': modelId,
    'modelNumber': modelNumber,
    'size': size,
    'color': color,
    'rgb': rgb,
    'measurements': measurements,
    'productBlueprintPatch': productBlueprintPatch,
    'token': token?.toJson(),
    'owner': owner?.toJson(),
  };
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

/// ✅ NEW: response model for
/// POST /mall/me/orders/scan/transfer
///
/// Backend returns:
/// { "data": { avatarId, productId, matched, txSignature?, fromWallet?, toWallet?, updatedToAddress? } }
class MallScanTransferResponse {
  MallScanTransferResponse({
    required this.avatarId,
    required this.productId,
    required this.matched,
    this.txSignature = '',
    this.fromWallet = '',
    this.toWallet = '',
    this.updatedToAddress = false,
  });

  final String avatarId;
  final String productId;

  final bool matched;

  // Transfer result
  final String txSignature;

  // Optional debug/info
  final String fromWallet;
  final String toWallet;

  // tokens/{productId}.toAddress updated?
  final bool updatedToAddress;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static bool _b(dynamic v) {
    if (v == null) return false;
    if (v is bool) return v;
    if (v is num) return v != 0;
    final s = v.toString().trim().toLowerCase();
    return s == 'true' || s == '1' || s == 'yes';
  }

  factory MallScanTransferResponse.fromJson(Map<String, dynamic> j) {
    // unwrap { data: ... } if exists (same pattern as other models)
    final maybeData = j['data'];
    if (maybeData is Map<String, dynamic>) {
      return MallScanTransferResponse.fromJson(maybeData);
    }
    if (maybeData is Map) {
      return MallScanTransferResponse.fromJson(
        Map<String, dynamic>.from(maybeData),
      );
    }

    return MallScanTransferResponse(
      avatarId: _s(j['avatarId']),
      productId: _s(j['productId']),
      matched: _b(j['matched']),
      txSignature: _s(j['txSignature']),
      fromWallet: _s(j['fromWallet']),
      toWallet: _s(j['toWallet']),
      updatedToAddress: _b(j['updatedToAddress']),
    );
  }

  Map<String, dynamic> toJson() => {
    'avatarId': avatarId,
    'productId': productId,
    'matched': matched,
    'txSignature': txSignature,
    'fromWallet': fromWallet,
    'toWallet': toWallet,
    'updatedToAddress': updatedToAddress,
  };
}

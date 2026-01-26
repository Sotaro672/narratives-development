// frontend/mall/lib/features/wallet/infrastructure/token_resolve_dto.dart

class SignedTokenContentFileDTO {
  const SignedTokenContentFileDTO({
    required this.type,
    required this.publicUri,
    required this.viewUri,
    this.viewExpiresAt,
  });

  final String type;

  /// 安定識別子（署名なしの場合は 403 になり得る）
  final String publicUri;

  /// ✅ 画面で使うべき URI（署名付き GET URL）
  final String viewUri;

  /// ISO8601 を想定（サーバは time.Time を返す）
  final String? viewExpiresAt;

  static String _asString(dynamic v) {
    if (v == null) return '';
    if (v is String) return v.trim();
    return v.toString().trim();
  }

  factory SignedTokenContentFileDTO.fromJson(Map<String, dynamic> json) {
    return SignedTokenContentFileDTO(
      type: _asString(json['type']),
      publicUri: _asString(json['publicUri']),
      viewUri: _asString(json['viewUri']),
      viewExpiresAt: _asString(json['viewExpiresAt']).isNotEmpty
          ? _asString(json['viewExpiresAt'])
          : null,
    );
  }

  Map<String, dynamic> toJson() => {
    'type': type,
    'publicUri': publicUri,
    'viewUri': viewUri,
    'viewExpiresAt': viewExpiresAt,
  };
}

class TokenResolveDTO {
  TokenResolveDTO({
    required this.productId,
    required this.brandId,
    required this.metadataUri,
    required this.mintAddress,
    this.brandName = '',
    this.productName = '',

    // ✅ NEW (required by updated backend response)
    this.tokenBlueprintId = '',
    this.tokenContentsFiles = const <SignedTokenContentFileDTO>[],
  });

  final String productId;
  final String brandId;
  final String metadataUri;
  final String mintAddress;

  // ✅ NEW: server may return empty when not resolved (non-fatal)
  final String brandName;
  final String productName;

  // ✅ NEW: metadata.attributes.TokenBlueprintID と同義（サーバが返す）
  final String tokenBlueprintId;

  // ✅ NEW: token-contents (signed GET URLs)
  final List<SignedTokenContentFileDTO> tokenContentsFiles;

  static String _asString(dynamic v) {
    if (v == null) return '';
    if (v is String) return v.trim();
    return v.toString().trim();
  }

  static List<SignedTokenContentFileDTO> _asFiles(dynamic v) {
    final out = <SignedTokenContentFileDTO>[];
    if (v is List) {
      for (final e in v) {
        if (e is Map<String, dynamic>) {
          out.add(SignedTokenContentFileDTO.fromJson(e));
        } else if (e is Map) {
          out.add(
            SignedTokenContentFileDTO.fromJson(Map<String, dynamic>.from(e)),
          );
        }
      }
    }
    // 念のため ".keep" を除外（サーバ側でも除外済み想定）
    return out
        .where((f) {
          final p = f.publicUri.trim();
          final v2 = f.viewUri.trim();
          return !(p.endsWith('/.keep') ||
              p.endsWith('.keep') ||
              v2.endsWith('/.keep') ||
              v2.endsWith('.keep'));
        })
        .toList(growable: false);
  }

  factory TokenResolveDTO.fromJson(Map<String, dynamic> json) {
    return TokenResolveDTO(
      productId: _asString(json['productId']),
      brandId: _asString(json['brandId']),
      brandName: _asString(json['brandName']),
      productName: _asString(json['productName']),
      metadataUri: _asString(json['metadataUri']),
      mintAddress: _asString(json['mintAddress']),

      // ✅ NEW
      tokenBlueprintId: _asString(json['tokenBlueprintId']),
      tokenContentsFiles: _asFiles(json['tokenContentsFiles']),
    );
  }

  Map<String, dynamic> toJson() => {
    'productId': productId,
    'brandId': brandId,
    'brandName': brandName,
    'productName': productName,
    'metadataUri': metadataUri,
    'mintAddress': mintAddress,

    // ✅ NEW
    'tokenBlueprintId': tokenBlueprintId,
    'tokenContentsFiles': tokenContentsFiles.map((e) => e.toJson()).toList(),
  };

  /// UI で使う“代表 contents URL”
  /// - 複数ファイルがあり得るため、まず viewUri を返す（あれば）
  /// - なければ空文字
  String get primaryContentsViewUrl {
    for (final f in tokenContentsFiles) {
      final u = f.viewUri.trim();
      if (u.isNotEmpty) return u;
    }
    return '';
  }
}

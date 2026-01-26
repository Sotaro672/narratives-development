// frontend/mall/lib/features/wallet/infrastructure/token_metadata_dto.dart

class TokenMetadataAttributeDTO {
  const TokenMetadataAttributeDTO({
    required this.traitType,
    required this.value,
  });

  final String traitType;
  final String value;

  factory TokenMetadataAttributeDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final trait = s(json['trait_type']).isNotEmpty
        ? s(json['trait_type'])
        : s(json['traitType']);
    final val = s(json['value']);
    return TokenMetadataAttributeDTO(traitType: trait, value: val);
  }
}

class TokenMetadataFileDTO {
  const TokenMetadataFileDTO({required this.type, required this.uri});

  final String type;
  final String uri;

  factory TokenMetadataFileDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();
    return TokenMetadataFileDTO(type: s(json['type']), uri: s(json['uri']));
  }
}

class TokenMetadataPropertiesDTO {
  const TokenMetadataPropertiesDTO({
    required this.category,
    required this.files,
  });

  final String category;
  final List<TokenMetadataFileDTO> files;

  factory TokenMetadataPropertiesDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final filesRaw = json['files'];
    final files = <TokenMetadataFileDTO>[];
    if (filesRaw is List) {
      for (final e in filesRaw) {
        if (e is Map<String, dynamic>) {
          files.add(TokenMetadataFileDTO.fromJson(e));
        } else if (e is Map) {
          files.add(
            TokenMetadataFileDTO.fromJson(Map<String, dynamic>.from(e)),
          );
        }
      }
    }

    return TokenMetadataPropertiesDTO(
      category: s(json['category']),
      files: files,
    );
  }
}

class TokenMetadataDTO {
  const TokenMetadataDTO({
    required this.name,
    required this.symbol,
    required this.description,
    required this.image,
    required this.externalUrl,
    required this.attributes,
    required this.createdAt,
    required this.properties,
  });

  final String name;
  final String symbol;
  final String description;

  /// 既存互換: これまで通り image も保持
  final String image;

  final String externalUrl;
  final List<TokenMetadataAttributeDTO> attributes;

  /// JSON: created_at
  final String createdAt;

  /// JSON: properties (category, files)
  final TokenMetadataPropertiesDTO? properties;

  factory TokenMetadataDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    final attrsRaw = json['attributes'];
    final attrs = <TokenMetadataAttributeDTO>[];
    if (attrsRaw is List) {
      for (final e in attrsRaw) {
        if (e is Map<String, dynamic>) {
          attrs.add(TokenMetadataAttributeDTO.fromJson(e));
        } else if (e is Map) {
          attrs.add(
            TokenMetadataAttributeDTO.fromJson(Map<String, dynamic>.from(e)),
          );
        }
      }
    }

    TokenMetadataPropertiesDTO? props;
    final propsRaw = json['properties'];
    if (propsRaw is Map<String, dynamic>) {
      props = TokenMetadataPropertiesDTO.fromJson(propsRaw);
    } else if (propsRaw is Map) {
      props = TokenMetadataPropertiesDTO.fromJson(
        Map<String, dynamic>.from(propsRaw),
      );
    }

    return TokenMetadataDTO(
      name: s(json['name']),
      symbol: s(json['symbol']),
      description: s(json['description']),
      image: s(json['image']),
      externalUrl: s(json['external_url']).isNotEmpty
          ? s(json['external_url'])
          : s(json['externalUrl']),
      attributes: attrs,
      createdAt: s(json['created_at']).isNotEmpty
          ? s(json['created_at'])
          : s(json['createdAt']),
      properties: props,
    );
  }

  /// TokenBlueprintID を metadata.attributes から抽出する（任意だが便利）
  String get tokenBlueprintId {
    for (final a in attributes) {
      if (a.traitType.trim() == 'TokenBlueprintID') {
        return a.value.trim();
      }
    }
    return '';
  }

  /// 取得したい“全リソースURI”を返す（image + properties.files[*].uri）
  /// - 重複は排除
  /// - 空文字は除外
  List<String> get allResourceUris {
    final set = <String>{};
    if (image.trim().isNotEmpty) set.add(image.trim());
    final files = properties?.files ?? const <TokenMetadataFileDTO>[];
    for (final f in files) {
      final u = f.uri.trim();
      if (u.isNotEmpty) set.add(u);
    }
    return set.toList(growable: false);
  }

  /// narratives-development_token_icon 側（優先: image、次点: properties.files の image/*）
  String? get tokenIconUri {
    if (image.trim().isNotEmpty) return image.trim();

    final files = properties?.files ?? const <TokenMetadataFileDTO>[];
    for (final f in files) {
      if (f.type.startsWith('image/')) {
        final u = f.uri.trim();
        if (u.isNotEmpty) return u;
      }
    }
    return null;
  }

  /// 旧来: narratives-development-token-contents 側の“素URI”を返す
  /// 注意:
  /// - bucket が private の場合、この URI はブラウザ/アプリから直接 GET できないことがある（403）。
  /// - 今後は server が返す tokenContentsFiles[*].viewUri（署名付き）を優先すること。
  String? get tokenContentsUri {
    final files = properties?.files ?? const <TokenMetadataFileDTO>[];
    for (final f in files) {
      if (f.type == 'application/octet-stream') {
        final u = f.uri.trim();
        if (u.isNotEmpty) return u;
      }
    }
    return null;
  }
}

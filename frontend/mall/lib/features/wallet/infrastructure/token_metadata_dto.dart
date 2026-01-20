//frontend\mall\lib\features\wallet\infrastructure\token_metadata_dto.dart
class TokenMetadataAttributeDTO {
  const TokenMetadataAttributeDTO({
    required this.traitType,
    required this.value,
  });

  final String traitType;
  final String value;

  factory TokenMetadataAttributeDTO.fromJson(Map<String, dynamic> json) {
    String s(dynamic v) => (v ?? '').toString().trim();

    // 一般的に "trait_type" / "traitType" が混在します
    final trait = s(json['trait_type']).isNotEmpty
        ? s(json['trait_type'])
        : s(json['traitType']);
    final val = s(json['value']);
    return TokenMetadataAttributeDTO(traitType: trait, value: val);
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
  });

  final String name;
  final String symbol;
  final String description;
  final String image;
  final String externalUrl;
  final List<TokenMetadataAttributeDTO> attributes;

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

    return TokenMetadataDTO(
      name: s(json['name']),
      symbol: s(json['symbol']),
      description: s(json['description']),
      image: s(json['image']),
      externalUrl: s(json['external_url']).isNotEmpty
          ? s(json['external_url'])
          : s(json['externalUrl']),
      attributes: attrs,
    );
  }
}

// frontend/mall/lib/features/wallet/infrastructure/token_resolve_dto.dart
class TokenResolveDTO {
  TokenResolveDTO({
    required this.productId,
    required this.brandId,
    required this.metadataUri,
    required this.mintAddress,
  });

  final String productId;
  final String brandId;
  final String metadataUri;
  final String mintAddress;

  static String _asString(dynamic v) {
    if (v == null) return '';
    if (v is String) return v.trim();
    return v.toString().trim();
  }

  factory TokenResolveDTO.fromJson(Map<String, dynamic> json) {
    return TokenResolveDTO(
      productId: _asString(json['productId']),
      brandId: _asString(json['brandId']),
      metadataUri: _asString(json['metadataUri']),
      mintAddress: _asString(json['mintAddress']),
    );
  }

  Map<String, dynamic> toJson() => {
    'productId': productId,
    'brandId': brandId,
    'metadataUri': metadataUri,
    'mintAddress': mintAddress,
  };
}

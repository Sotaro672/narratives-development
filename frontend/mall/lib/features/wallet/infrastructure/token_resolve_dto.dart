// frontend/mall/lib/features/wallet/infrastructure/token_resolve_dto.dart
class TokenResolveDTO {
  TokenResolveDTO({
    required this.productId,
    required this.brandId,
    required this.metadataUri,
    required this.mintAddress,
    this.brandName = '',
    this.productName = '',
  });

  final String productId;
  final String brandId;
  final String metadataUri;
  final String mintAddress;

  // ✅ NEW: server may return empty when not resolved (non-fatal)
  final String brandName;
  final String productName;

  static String _asString(dynamic v) {
    if (v == null) return '';
    if (v is String) return v.trim();
    return v.toString().trim();
  }

  factory TokenResolveDTO.fromJson(Map<String, dynamic> json) {
    return TokenResolveDTO(
      productId: _asString(json['productId']),
      brandId: _asString(json['brandId']),
      brandName: _asString(json['brandName']), // ✅ NEW
      productName: _asString(json['productName']), // ✅ NEW
      metadataUri: _asString(json['metadataUri']),
      mintAddress: _asString(json['mintAddress']),
    );
  }

  Map<String, dynamic> toJson() => {
    'productId': productId,
    'brandId': brandId,
    'brandName': brandName, // ✅ NEW
    'productName': productName, // ✅ NEW
    'metadataUri': metadataUri,
    'mintAddress': mintAddress,
  };
}

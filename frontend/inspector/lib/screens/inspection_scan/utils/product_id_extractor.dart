// lib/screens/inspection_scan/utils/product_id_extractor.dart
String? extractProductIdFromQrRaw(String raw) {
  final trimmed = raw.trim();
  if (trimmed.isEmpty) return null;

  // 正規のURLとして解釈できる場合
  try {
    final uri = Uri.parse(trimmed);
    if (uri.scheme == 'http' || uri.scheme == 'https') {
      final segments = uri.pathSegments.where((s) => s.isNotEmpty).toList();
      if (segments.isNotEmpty) {
        return segments.last;
      }
    }
  } catch (_) {
    // 無視してフォールバックへ
  }

  // URLっぽい文字列のフォールバック
  if (trimmed.contains('https://') || trimmed.contains('http://')) {
    final lastSlash = trimmed.lastIndexOf('/');
    if (lastSlash != -1 && lastSlash + 1 < trimmed.length) {
      return trimmed.substring(lastSlash + 1);
    }
  }

  // それ以外はそのままID扱い
  return trimmed;
}

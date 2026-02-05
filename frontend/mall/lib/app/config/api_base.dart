// frontend\mall\lib\app\config\api_base.dart
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

/// ✅ Common API base resolver
/// Priority:
/// 1) --dart-define=API_BASE_URL=https://...
/// 2) --dart-define=API_BASE=https://...      (backward compatible)
/// 3) fallback
String resolveApiBase() => _resolveApiBase();

String _resolveApiBase() {
  const fromDefineUrl = String.fromEnvironment('API_BASE_URL');
  const fromDefine = String.fromEnvironment('API_BASE');

  final v = fromDefineUrl.trim().isNotEmpty
      ? fromDefineUrl.trim()
      : fromDefine.trim().isNotEmpty
      ? fromDefine.trim()
      : _fallbackBaseUrl.trim();

  // normalize: remove trailing slashes
  return v.replaceAll(RegExp(r'\/+$'), '');
}

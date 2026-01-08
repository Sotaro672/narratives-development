// frontend\mall\lib\app\config\api_base.dart
const String _fallbackBaseUrl =
    'https://narratives-backend-871263659099.asia-northeast1.run.app';

/// ✅ make this public within library (no underscore) so other files can reuse
/// the exact same resolution logic without duplicating fallback constants.
String resolveSnsApiBase() => _resolveApiBase();

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

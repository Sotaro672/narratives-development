// frontend/mall/lib/app/routing/avatar_name_store.dart
import 'package:flutter/foundation.dart';

/// Global store for "avatarName" resolved from backend (/mall/me/avatar).
/// - App header title should prefer this over FirebaseAuth displayName/email.
/// - Updated by use_avatar.dart after fetching MeAvatar.
///
/// NOTE:
/// - Keep it simple: in-memory only.
/// - Router refreshListenable will rebuild AppShell title when this changes.
class AvatarNameStore extends ChangeNotifier {
  AvatarNameStore._();
  static final AvatarNameStore I = AvatarNameStore._();

  String _avatarName = '';

  String get avatarName => _avatarName;

  bool get hasAvatarName => _avatarName.trim().isNotEmpty;

  void setAvatarName(String? name) {
    final next = (name ?? '').trim();
    if (next == _avatarName) return;
    _avatarName = next;
    notifyListeners();
  }

  void clear() {
    if (_avatarName.isEmpty) return;
    _avatarName = '';
    notifyListeners();
  }
}

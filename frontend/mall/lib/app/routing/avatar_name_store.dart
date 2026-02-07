// frontend/mall/lib/app/routing/avatar_name_store.dart
import 'package:flutter/foundation.dart';

/// Global store for "avatarName" resolved from backend (/mall/me/avatars).
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

  // ----------------------------------------
  // Logging helper (Webでも確実に出す)
  // ----------------------------------------
  void _log(String msg) {
    final out = '[AvatarNameStore] $msg';
    debugPrint(out); // Flutter logs
    // ignore: avoid_print
    print(out); // Web console fallback
  }

  void setAvatarName(String? name) {
    final raw = name;
    final next = (name ?? '').trim();
    final prev = _avatarName;

    _log('setAvatarName called raw="${raw ?? ''}" next="$next" prev="$prev"');

    if (next == prev) {
      _log('setAvatarName noop (no change)');
      return;
    }

    _avatarName = next;
    _log(
      'setAvatarName updated avatarName="$_avatarName" -> notifyListeners()',
    );
    notifyListeners();
  }

  void clear() {
    _log('clear called prev="$_avatarName"');

    if (_avatarName.isEmpty) {
      _log('clear noop (already empty)');
      return;
    }

    _avatarName = '';
    _log('clear updated avatarName="" -> notifyListeners()');
    notifyListeners();
  }
}

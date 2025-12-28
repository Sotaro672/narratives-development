// frontend/sns/lib/features/auth/presentation/hook/use_shipping_address.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../user/infrastructure/user_repository_http.dart';
import '../../../shippingAddress/infrastructure/shipping_address_repository_http.dart';

/// ShippingAddressPage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseShippingAddress extends ChangeNotifier {
  UseShippingAddress({
    required this.mode,
    required this.oobCode,
    required this.continueUrl,
    required this.lang,
    required this.from,
    required this.intent,
  });

  // Firebase action params
  final String? mode; // e.g. verifyEmail
  final String? oobCode;
  final String? continueUrl;
  final String? lang;

  // optional app params
  final String? from;
  final String? intent;

  // ============================================================
  // API base (match list_repository_http.dart behavior)
  // ============================================================
  static const String _fallbackBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  String _resolveApiBase() {
    const fromDefine = String.fromEnvironment('API_BASE_URL');
    final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
    return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
  }

  // ============================================================
  // repositories
  // ============================================================
  late final UserRepositoryHttp _userRepo;
  late final ShippingAddressRepositoryHttp _shipRepo;

  bool _initialized = false;

  // ============================================================
  // verifying state
  // ============================================================
  bool verifying = false;
  bool verified = false;
  String? verifyError;

  // ---- profile form ----
  final lastNameCtrl = TextEditingController();
  final lastNameKanaCtrl = TextEditingController();
  final firstNameCtrl = TextEditingController();
  final firstNameKanaCtrl = TextEditingController();

  // ---- address form ----
  final zipCtrl = TextEditingController();
  final prefCtrl = TextEditingController();
  final cityCtrl = TextEditingController();
  final addr1Ctrl = TextEditingController();
  final addr2Ctrl = TextEditingController();

  bool zipLoading = false;
  String? zipError;

  bool saving = false;
  String? saveMsg;

  String _s(String? v) => (v ?? '').trim();

  void _log(String msg) {
    if (!kDebugMode) return;
    debugPrint(msg);
  }

  /// Page initState から呼ぶ
  void init() {
    if (_initialized) return;
    _initialized = true;

    final baseUrl = _resolveApiBase();
    _userRepo = UserRepositoryHttp(baseUrl: baseUrl);
    _shipRepo = ShippingAddressRepositoryHttp(baseUrl: baseUrl);

    // ✅ verifyEmail の場合は actionCode 適用
    maybeApplyActionCode();

    // ✅ 郵便番号が変わったら自動で検索（7桁になったタイミング）
    zipCtrl.addListener(_onZipChanged);

    // ✅ ボタン enable/disable を即時反映
    lastNameCtrl.addListener(_onFormChanged);
    lastNameKanaCtrl.addListener(_onFormChanged);
    firstNameCtrl.addListener(_onFormChanged);
    firstNameKanaCtrl.addListener(_onFormChanged);

    prefCtrl.addListener(_onFormChanged);
    cityCtrl.addListener(_onFormChanged);
    addr1Ctrl.addListener(_onFormChanged);
    addr2Ctrl.addListener(_onFormChanged);
  }

  @override
  void dispose() {
    zipCtrl.removeListener(_onZipChanged);

    lastNameCtrl.removeListener(_onFormChanged);
    lastNameKanaCtrl.removeListener(_onFormChanged);
    firstNameCtrl.removeListener(_onFormChanged);
    firstNameKanaCtrl.removeListener(_onFormChanged);

    prefCtrl.removeListener(_onFormChanged);
    cityCtrl.removeListener(_onFormChanged);
    addr1Ctrl.removeListener(_onFormChanged);
    addr2Ctrl.removeListener(_onFormChanged);

    lastNameCtrl.dispose();
    lastNameKanaCtrl.dispose();
    firstNameCtrl.dispose();
    firstNameKanaCtrl.dispose();

    zipCtrl.dispose();
    prefCtrl.dispose();
    cityCtrl.dispose();
    addr1Ctrl.dispose();
    addr2Ctrl.dispose();

    if (_initialized) {
      _userRepo.dispose();
      _shipRepo.dispose();
    }

    super.dispose();
  }

  void _onFormChanged() {
    notifyListeners();
  }

  /// ✅ “戻る” の遷移先
  String backTo() {
    final f = _s(from);
    if (f.isNotEmpty) return f;

    final cu = _s(continueUrl);
    if (cu.isNotEmpty) {
      final uri = Uri.tryParse(cu);
      if (uri != null) {
        final path = uri.path.isEmpty ? '/' : uri.path;
        final query = uri.query;
        return query.isEmpty ? path : '$path?$query';
      }
    }
    return '/';
  }

  bool get loggedIn => FirebaseAuth.instance.currentUser != null;

  bool get emailVerified =>
      FirebaseAuth.instance.currentUser?.emailVerified ?? false;

  bool get cameFromEmailLink {
    final m = _s(mode);
    final o = _s(oobCode);
    return (m == 'verifyEmail' && o.isNotEmpty);
  }

  Future<void> maybeApplyActionCode() async {
    final m = _s(mode);
    final oob = _s(oobCode);

    // ✅ verifyEmail 以外 / oobCode なしは何もしない
    if (m != 'verifyEmail' || oob.isEmpty) return;

    verifying = true;
    verified = false;
    verifyError = null;
    notifyListeners();

    try {
      await FirebaseAuth.instance.applyActionCode(oob);

      final user = FirebaseAuth.instance.currentUser;
      if (user != null) {
        await user.reload();
      }

      verified = true;
      notifyListeners();
    } on FirebaseAuthException catch (e) {
      verifyError = _friendlyActionError(e);
      notifyListeners();
    } catch (e) {
      verifyError = e.toString();
      notifyListeners();
    } finally {
      verifying = false;
      notifyListeners();
    }
  }

  String _friendlyActionError(FirebaseAuthException e) {
    switch (e.code) {
      case 'expired-action-code':
        return 'この認証リンクは期限切れです。もう一度認証メールを送信してください。';
      case 'invalid-action-code':
        return 'この認証リンクは無効です。もう一度認証メールを送信してください。';
      case 'user-disabled':
        return 'このアカウントは無効化されています。';
      default:
        return e.message ?? 'メール認証に失敗しました。';
    }
  }

  // ----------------------------------------------------------------
  // 郵便番号 → 住所自動入力
  // ----------------------------------------------------------------

  String normalizeZip(String s) {
    return s.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String? _lastResolvedZip;

  void _onZipChanged() {
    final zip = normalizeZip(zipCtrl.text);

    if (zip.length == 7 && zip != _lastResolvedZip) {
      _lastResolvedZip = zip;
      lookupZipAndFill(zip);
    } else {
      if (zipError != null) {
        zipError = null;
        notifyListeners();
      }
    }

    notifyListeners();
  }

  Future<void> onZipSearchPressed() async {
    final zip = normalizeZip(zipCtrl.text);
    if (zip.length == 7) {
      _lastResolvedZip = zip;
      await lookupZipAndFill(zip);
      return;
    }
    zipError = '郵便番号は7桁で入力してください。';
    notifyListeners();
  }

  Future<void> lookupZipAndFill(String zip7) async {
    if (zipLoading) return;

    zipLoading = true;
    zipError = null;
    notifyListeners();

    try {
      final uri = Uri.parse(
        'https://zipcloud.ibsnet.co.jp/api/search?zipcode=$zip7',
      );
      final res = await http.get(uri);

      if (res.statusCode != 200) {
        throw StateError('住所検索に失敗しました（HTTP ${res.statusCode}）。');
      }

      final json = jsonDecode(res.body) as Map<String, dynamic>;
      final status = json['status'];
      if (status != 200) {
        final msg = (json['message'] ?? '住所検索に失敗しました。').toString();
        throw StateError(msg);
      }

      final results = json['results'];
      if (results == null) {
        final msg = (json['message'] ?? '該当する住所が見つかりませんでした。').toString();
        throw StateError(msg);
      }

      final list = results as List<dynamic>;
      if (list.isEmpty) {
        throw StateError('該当する住所が見つかりませんでした。');
      }

      final r0 = list.first as Map<String, dynamic>;
      final pref = (r0['address1'] ?? '').toString();
      final city = (r0['address2'] ?? '').toString();
      final town = (r0['address3'] ?? '').toString();

      prefCtrl.text = pref;
      cityCtrl.text = city;
      addr1Ctrl.text = town;

      zipError = null;
      notifyListeners();
    } catch (e) {
      zipError = e.toString();
      notifyListeners();
    } finally {
      zipLoading = false;
      notifyListeners();
    }
  }

  // ----------------------------------------------------------------
  // Save (USER + SHIPPING_ADDRESS)
  // ----------------------------------------------------------------

  bool get canSaveAddress {
    final fbUser = FirebaseAuth.instance.currentUser;
    final loggedIn = fbUser != null;

    return !saving &&
        loggedIn &&
        _s(lastNameCtrl.text).isNotEmpty &&
        _s(firstNameCtrl.text).isNotEmpty &&
        normalizeZip(zipCtrl.text).isNotEmpty &&
        _s(prefCtrl.text).isNotEmpty &&
        _s(cityCtrl.text).isNotEmpty &&
        _s(addr1Ctrl.text).isNotEmpty;
  }

  Future<void> saveAddressToBackend(BuildContext context) async {
    final fbUser = FirebaseAuth.instance.currentUser;
    if (fbUser == null) {
      saveMsg = 'サインインが必要です。';
      notifyListeners();
      return;
    }

    final uid = fbUser.uid.trim();
    if (uid.isEmpty) {
      saveMsg = 'uid が取得できませんでした。';
      notifyListeners();
      return;
    }

    // ✅ 4つの入力値
    final lastName = _s(lastNameCtrl.text);
    final lastNameKana = _s(lastNameKanaCtrl.text);
    final firstName = _s(firstNameCtrl.text);
    final firstNameKana = _s(firstNameKanaCtrl.text);

    final zip7 = normalizeZip(zipCtrl.text);
    final pref = _s(prefCtrl.text);
    final city = _s(cityCtrl.text);
    final addr1 = _s(addr1Ctrl.text);
    final addr2 = _s(addr2Ctrl.text);

    saveMsg = null;
    saving = true;
    notifyListeners();

    bool userSaved = false;
    String? userErr;

    try {
      // ----------------------------
      // 1) upsert user (best-effort)
      // ----------------------------
      _log(
        '[ShippingAddressPage] upsert user uid=$uid '
        'lastName="$lastName" lastNameKana="$lastNameKana" '
        'firstName="$firstName" firstNameKana="$firstNameKana"',
      );

      try {
        await _userRepo.create(
          CreateUserBody(
            id: uid,
            firstName: firstName,
            firstNameKana: firstNameKana.isEmpty ? null : firstNameKana,
            lastName: lastName,
            lastNameKana: lastNameKana.isEmpty ? null : lastNameKana,
          ),
        );
        userSaved = true;
      } catch (e) {
        _log('[ShippingAddressPage] user create failed -> try update. err=$e');
        try {
          await _userRepo.update(
            uid,
            UpdateUserBody(
              firstName: firstName,
              firstNameKana: firstNameKana.isEmpty ? null : firstNameKana,
              lastName: lastName,
              lastNameKana: lastNameKana.isEmpty ? null : lastNameKana,
            ),
          );
          userSaved = true;
        } catch (e2) {
          userErr = e2.toString();
          _log('[ShippingAddressPage] user update failed. err=$e2');
        }
      }

      // ----------------------------
      // 2) upsert shipping address (required)
      // ----------------------------
      _log('[ShippingAddressPage] upsert shippingAddress userId=$uid');

      final created = await _tryUpsertShippingAddress(
        uid: uid,
        zip7: zip7,
        pref: pref,
        city: city,
        addr1: addr1,
        addr2: addr2,
      );

      final msg = StringBuffer()
        ..writeln('配送先情報を保存しました。')
        ..writeln('shippingAddressId=${created.id} userId=${created.userId}');
      if (userSaved) {
        msg.writeln('user: saved');
      } else if (userErr != null) {
        msg.writeln('user: failed (non-blocking) $userErr');
      }

      saveMsg = msg.toString().trim();
      notifyListeners();

      if (!context.mounted) return;
      context.go('/billing-address');
    } catch (e) {
      saveMsg = e.toString();
      notifyListeners();
    } finally {
      saving = false;
      notifyListeners();
    }
  }

  Future<ShippingAddress> _tryUpsertShippingAddress({
    required String uid,
    required String zip7,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    try {
      return await _shipRepo.create(
        CreateShippingAddressInput(
          userId: uid,
          zipCode: zip7,
          state: pref,
          city: city,
          street: addr1,
          street2: addr2.isEmpty ? null : addr2,
          country: 'JP',
        ),
      );
    } catch (e) {
      _log(
        '[ShippingAddressPage] shipping create failed -> try update. err=$e',
      );

      // NOTE: id=uid 前提（あなたの backend の設計に合わせて）
      return await _shipRepo.update(
        uid,
        UpdateShippingAddressInput(
          zipCode: zip7,
          state: pref,
          city: city,
          street: addr1,
          street2: addr2, // "" を渡すと消去扱いにできる
          country: 'JP',
        ),
      );
    }
  }

  void goSignIn(BuildContext context) {
    final from = Uri.encodeComponent(GoRouterState.of(context).uri.toString());
    context.go('/login?from=$from');
  }
}

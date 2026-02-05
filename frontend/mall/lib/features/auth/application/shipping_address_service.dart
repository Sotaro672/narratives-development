// frontend/mall/lib/features/auth/application/shipping_address_service.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import '../../../app/config/api_base.dart';

import '../../user/infrastructure/user_repository_http.dart';
import '../../shippingAddress/infrastructure/repository_http.dart';

class VerifyEmailState {
  const VerifyEmailState({
    required this.verifying,
    required this.verified,
    required this.error,
  });

  final bool verifying;
  final bool verified;
  final String? error;

  static const idle = VerifyEmailState(
    verifying: false,
    verified: false,
    error: null,
  );

  VerifyEmailState copyWith({bool? verifying, bool? verified, String? error}) {
    return VerifyEmailState(
      verifying: verifying ?? this.verifying,
      verified: verified ?? this.verified,
      error: error,
    );
  }
}

class ZipLookupResult {
  const ZipLookupResult.success({
    required this.pref,
    required this.city,
    required this.town,
  }) : ok = true,
       error = null;

  const ZipLookupResult.failure(this.error)
    : ok = false,
      pref = '',
      city = '',
      town = '';

  final bool ok;
  final String pref;
  final String city;
  final String town;
  final String? error;
}

class SaveAddressResult {
  const SaveAddressResult.success(this.message) : ok = true, error = null;

  const SaveAddressResult.failure(this.error) : ok = false, message = null;

  final bool ok;
  final String? message;
  final String? error;
}

class BasicResult {
  const BasicResult.success() : ok = true, error = null;

  const BasicResult.failure(this.error) : ok = false;

  final bool ok;
  final String? error;
}

class ShippingAddressService {
  ShippingAddressService({
    FirebaseAuth? auth,
    http.Client? httpClient,
    UserRepositoryHttp? userRepo,
    ShippingAddressRepositoryHttp? shipRepo,

    /// ✅ baseUrl は「ルート」(SNS/Mall 名を含めない) を渡す想定
    /// 例: https://...run.app
    String? baseUrl,
  }) : _auth = auth ?? FirebaseAuth.instance,
       _http = httpClient ?? http.Client(),
       _baseUrl = _normalizeBaseUrl((baseUrl ?? resolveApiBase()).trim()),
       _userRepo = userRepo,
       _shipRepo = shipRepo {
    // repo が未注入なら baseUrl で作る（repo 側が /mall/... を付与する想定）
    final b = _baseUrl;
    _userRepoInst = _userRepo ?? UserRepositoryHttp(baseUrl: b);
    _shipRepoInst = _shipRepo ?? ShippingAddressRepositoryHttp(baseUrl: b);
  }

  // SharedPreferences keys (NEW)
  static const _kPendingEmailForLinkSignIn = 'pendingEmailForLinkSignIn';

  final FirebaseAuth _auth;
  final http.Client _http;

  /// ✅ ルートURL（末尾スラッシュなし）
  final String _baseUrl;

  final UserRepositoryHttp? _userRepo;
  final ShippingAddressRepositoryHttp? _shipRepo;

  late final UserRepositoryHttp _userRepoInst;
  late final ShippingAddressRepositoryHttp _shipRepoInst;

  void dispose() {
    try {
      _userRepoInst.dispose();
    } catch (_) {}
    try {
      _shipRepoInst.dispose();
    } catch (_) {}
    try {
      _http.close();
    } catch (_) {}
  }

  String s(String? v) => (v ?? '').trim();

  void log(String msg) {
    if (!kDebugMode) return;
    debugPrint(msg);
  }

  // ------------------------------------------------------------
  // routing helpers
  // ------------------------------------------------------------

  String backTo({required String? from, required String? continueUrl}) {
    final f = s(from);
    if (f.isNotEmpty) return f;

    final cu = s(continueUrl);
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

  bool cameFromEmailLink({required String? mode, required String? oobCode}) {
    final m = s(mode);
    final o = s(oobCode);
    return (m == 'verifyEmail' && o.isNotEmpty);
  }

  bool get loggedIn => _auth.currentUser != null;

  bool get emailVerified => _auth.currentUser?.emailVerified ?? false;

  // ------------------------------------------------------------
  // NEW: Email link auto sign-in
  // ------------------------------------------------------------

  /// Save pending email for email-link sign-in (call this when you send the sign-in email).
  Future<void> setPendingEmailForLinkSignIn(String email) async {
    final e = s(email);
    if (e.isEmpty) return;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kPendingEmailForLinkSignIn, e);
  }

  Future<String?> _getPendingEmailForLinkSignIn() async {
    final prefs = await SharedPreferences.getInstance();
    final v = s(prefs.getString(_kPendingEmailForLinkSignIn));
    return v.isEmpty ? null : v;
  }

  Future<void> _clearPendingEmailForLinkSignIn() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_kPendingEmailForLinkSignIn);
  }

  /// Try to auto sign-in if [continueUrl] is an email sign-in link.
  /// - Works for "Email link sign-in (magic link)" flows.
  /// - If this is only "verifyEmail" link, this will usually do nothing (ok=true, no-op).
  Future<BasicResult> tryAutoSignInFromEmailLink({
    required String? continueUrl,
  }) async {
    // already signed in -> no-op
    if (_auth.currentUser != null) return const BasicResult.success();

    final link = s(continueUrl);
    if (link.isEmpty) {
      // no link to process -> no-op
      return const BasicResult.success();
    }

    try {
      final isLink = _auth.isSignInWithEmailLink(link);
      if (!isLink) {
        // not a sign-in link -> no-op (verifyEmail link etc.)
        return const BasicResult.success();
      }

      final email = await _getPendingEmailForLinkSignIn();
      if (email == null || email.isEmpty) {
        return const BasicResult.failure(
          'サインイン用メールアドレスが見つかりませんでした。もう一度サインインメールを送信してください。',
        );
      }

      log('[Auth] signInWithEmailLink link=$link email=$email');
      await _auth.signInWithEmailLink(email: email, emailLink: link);

      // 成功したら pending を消す（再利用防止）
      await _clearPendingEmailForLinkSignIn();

      // ensure currentUser refreshed
      final user = _auth.currentUser;
      if (user != null) {
        await user.reload();
      }

      return const BasicResult.success();
    } on FirebaseAuthException catch (e) {
      log('[Auth] signInWithEmailLink failed code=${e.code} msg=${e.message}');
      return BasicResult.failure(_friendlyLinkSignInError(e));
    } catch (e) {
      log('[Auth] signInWithEmailLink failed err=$e');
      return BasicResult.failure(e.toString());
    }
  }

  String _friendlyLinkSignInError(FirebaseAuthException e) {
    switch (e.code) {
      case 'expired-action-code':
      case 'invalid-action-code':
        return 'このリンクは期限切れ、または無効です。もう一度サインインメールを送信してください。';
      case 'user-disabled':
        return 'このアカウントは無効化されています。';
      default:
        return e.message ?? '自動サインインに失敗しました。';
    }
  }

  // ------------------------------------------------------------
  // verify email action code
  // ------------------------------------------------------------

  Future<VerifyEmailState> applyActionCodeIfNeeded({
    required String? mode,
    required String? oobCode,
  }) async {
    final m = s(mode);
    final oob = s(oobCode);

    // ✅ verifyEmail 以外 / oobCode なしは何もしない
    if (m != 'verifyEmail' || oob.isEmpty) {
      return VerifyEmailState.idle;
    }

    try {
      await _auth.applyActionCode(oob);

      final user = _auth.currentUser;
      if (user != null) {
        await user.reload();
      }

      return const VerifyEmailState(
        verifying: false,
        verified: true,
        error: null,
      );
    } on FirebaseAuthException catch (e) {
      return VerifyEmailState(
        verifying: false,
        verified: false,
        error: friendlyActionError(e),
      );
    } catch (e) {
      return VerifyEmailState(
        verifying: false,
        verified: false,
        error: e.toString(),
      );
    }
  }

  String friendlyActionError(FirebaseAuthException e) {
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

  // ------------------------------------------------------------
  // NEW: Ensure mall user (/mall/sign-in)
  // ------------------------------------------------------------

  Future<BasicResult> ensureMallUser() async {
    final user = _auth.currentUser;
    if (user == null) {
      return const BasicResult.failure('サインインが必要です。');
    }

    try {
      final idToken = await user.getIdToken();
      final uri = Uri.parse('$_baseUrl/mall/sign-in');

      log('[Mall] POST /mall/sign-in base=$_baseUrl uid=${user.uid}');

      final res = await _http.post(
        uri,
        headers: <String, String>{
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $idToken',
        },
        body: jsonEncode(<String, dynamic>{}),
      );

      if (res.statusCode >= 200 && res.statusCode < 300) {
        return const BasicResult.success();
      }

      // try parse backend error
      String msg = 'ユーザー初期化に失敗しました（HTTP ${res.statusCode}）。';
      try {
        final j = jsonDecode(res.body);
        if (j is Map && j['error'] != null) {
          msg = j['error'].toString();
        }
      } catch (_) {}

      log(
        '[Mall] /mall/sign-in failed status=${res.statusCode} body=${res.body}',
      );
      return BasicResult.failure(msg);
    } catch (e) {
      log('[Mall] /mall/sign-in failed err=$e');
      return BasicResult.failure(e.toString());
    }
  }

  // ------------------------------------------------------------
  // zip lookup
  // ------------------------------------------------------------

  String normalizeZip(String input) => input.replaceAll(RegExp(r'[^0-9]'), '');

  Future<ZipLookupResult> lookupZip(String zip7) async {
    try {
      final uri = Uri.parse(
        'https://zipcloud.ibsnet.co.jp/api/search?zipcode=$zip7',
      );
      final res = await _http.get(uri);

      if (res.statusCode != 200) {
        return ZipLookupResult.failure('住所検索に失敗しました（HTTP ${res.statusCode}）。');
      }

      final json = jsonDecode(res.body) as Map<String, dynamic>;
      final status = json['status'];
      if (status != 200) {
        final msg = (json['message'] ?? '住所検索に失敗しました。').toString();
        return ZipLookupResult.failure(msg);
      }

      final results = json['results'];
      if (results == null) {
        final msg = (json['message'] ?? '該当する住所が見つかりませんでした。').toString();
        return ZipLookupResult.failure(msg);
      }

      final list = results as List<dynamic>;
      if (list.isEmpty) {
        return const ZipLookupResult.failure('該当する住所が見つかりませんでした。');
      }

      final r0 = list.first as Map<String, dynamic>;
      final pref = (r0['address1'] ?? '').toString();
      final city = (r0['address2'] ?? '').toString();
      final town = (r0['address3'] ?? '').toString();

      return ZipLookupResult.success(pref: pref, city: city, town: town);
    } catch (e) {
      return ZipLookupResult.failure(e.toString());
    }
  }

  // ------------------------------------------------------------
  // save
  // ------------------------------------------------------------

  bool canSaveAddress({
    required bool saving,
    required String lastName,
    required String firstName,
    required String zip,
    required String pref,
    required String city,
    required String addr1,
  }) {
    final fbUser = _auth.currentUser;
    final isLoggedIn = fbUser != null;

    return !saving &&
        isLoggedIn &&
        s(lastName).isNotEmpty &&
        s(firstName).isNotEmpty &&
        normalizeZip(zip).isNotEmpty &&
        s(pref).isNotEmpty &&
        s(city).isNotEmpty &&
        s(addr1).isNotEmpty;
  }

  Future<SaveAddressResult> saveAddress({
    required String lastName,
    required String lastNameKana,
    required String firstName,
    required String firstNameKana,
    required String zip,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    final fbUser = _auth.currentUser;
    if (fbUser == null) {
      return const SaveAddressResult.failure('サインインが必要です。');
    }

    final uid = fbUser.uid.trim();
    if (uid.isEmpty) {
      return const SaveAddressResult.failure('uid が取得できませんでした。');
    }

    final ln = s(lastName);
    final lnk = s(lastNameKana);
    final fn = s(firstName);
    final fnk = s(firstNameKana);

    final zip7 = normalizeZip(zip);
    final st = s(pref);
    final ct = s(city);
    final a1 = s(addr1);
    final a2 = s(addr2);

    bool userSaved = false;
    String? userErr;

    try {
      // ----------------------------
      // 1) upsert user (best-effort)
      // ----------------------------
      log(
        '[ShippingAddress] upsert user uid=$uid '
        'lastName="$ln" lastNameKana="$lnk" firstName="$fn" firstNameKana="$fnk"',
      );

      try {
        await _userRepoInst.create(
          CreateUserBody(
            id: uid,
            firstName: fn,
            firstNameKana: fnk.isEmpty ? null : fnk,
            lastName: ln,
            lastNameKana: lnk.isEmpty ? null : lnk,
          ),
        );
        userSaved = true;
      } catch (e) {
        log('[ShippingAddress] user create failed -> try update. err=$e');
        try {
          await _userRepoInst.update(
            uid,
            UpdateUserBody(
              firstName: fn,
              firstNameKana: fnk.isEmpty ? null : fnk,
              lastName: ln,
              lastNameKana: lnk.isEmpty ? null : lnk,
            ),
          );
          userSaved = true;
        } catch (e2) {
          userErr = e2.toString();
          log('[ShippingAddress] user update failed. err=$e2');
        }
      }

      // ----------------------------
      // 2) upsert shipping address (required)
      // ----------------------------
      log('[ShippingAddress] upsert shippingAddress uid=$uid');

      final saved = await upsertShippingAddress(
        uid: uid,
        zip7: zip7,
        pref: st,
        city: ct,
        addr1: a1,
        addr2: a2,
      );

      final sb = StringBuffer()
        ..writeln('配送先情報を保存しました。')
        ..writeln('shippingAddressId=${saved.id} userId=${saved.userId}');
      if (userSaved) {
        sb.writeln('user: saved');
      } else if (userErr != null) {
        sb.writeln('user: failed (non-blocking) $userErr');
      }

      return SaveAddressResult.success(sb.toString().trim());
    } catch (e) {
      return SaveAddressResult.failure(e.toString());
    }
  }

  /// ✅ docId=uid 前提の upsert（avatarId 不要）
  Future<ShippingAddress> upsertShippingAddress({
    required String uid,
    required String zip7,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    log(
      '[ShippingAddress] upsertShippingAddress '
      'id(uid)=$uid zip=$zip7 state="$pref" city="$city" street="$addr1" street2="$addr2"',
    );

    // ✅ UpsertShippingAddressInput を使う（id=userId=uid）
    return await _shipRepoInst.create(
      UpsertShippingAddressInput(
        id: uid,
        userId: uid,
        zipCode: zip7,
        state: pref,
        city: city,
        street: addr1,
        street2: addr2.isEmpty ? null : addr2,
        country: 'JP',
      ),
    );
  }

  // ------------------------------------------------------------
  // helpers
  // ------------------------------------------------------------

  static String _normalizeBaseUrl(String v) {
    final s = v.trim();
    if (s.isEmpty) return '';
    // normalize: remove trailing slashes
    return s.replaceAll(RegExp(r'\/+$'), '');
  }
}

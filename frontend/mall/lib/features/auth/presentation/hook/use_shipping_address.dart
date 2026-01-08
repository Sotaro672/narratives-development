// frontend\mall\lib\features\auth\presentation\hook\use_shipping_address.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../application/shipping_address_service.dart';

/// ShippingAddressPage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseShippingAddress extends ChangeNotifier {
  UseShippingAddress({
    required this.mode,
    required this.oobCode,
    required this.continueUrl,
    required this.lang,
    required this.from,
    required this.intent,
    ShippingAddressService? service,
  }) : _service = service ?? ShippingAddressService();

  // Firebase action params
  final String? mode; // e.g. verifyEmail
  final String? oobCode;
  final String? continueUrl;
  final String? lang;

  // optional app params
  final String? from;
  final String? intent;

  final ShippingAddressService _service;

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

  /// Page initState から呼ぶ
  void init() {
    if (_initialized) return;
    _initialized = true;

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

    _service.dispose();

    super.dispose();
  }

  void _onFormChanged() {
    notifyListeners();
  }

  // ------------------------------------------------------------
  // computed (delegate)
  // ------------------------------------------------------------

  String backTo() => _service.backTo(from: from, continueUrl: continueUrl);

  bool get loggedIn => _service.loggedIn;

  bool get emailVerified => _service.emailVerified;

  bool get cameFromEmailLink =>
      _service.cameFromEmailLink(mode: mode, oobCode: oobCode);

  String normalizeZip(String s) => _service.normalizeZip(s);

  bool get canSaveAddress => _service.canSaveAddress(
    saving: saving,
    lastName: lastNameCtrl.text,
    firstName: firstNameCtrl.text,
    zip: zipCtrl.text,
    pref: prefCtrl.text,
    city: cityCtrl.text,
    addr1: addr1Ctrl.text,
  );

  // ------------------------------------------------------------
  // verify email (delegate)
  // ------------------------------------------------------------

  Future<void> maybeApplyActionCode() async {
    final m = (mode ?? '').trim();
    final o = (oobCode ?? '').trim();

    // ✅ verifyEmail 以外 / oobCode なしは何もしない
    if (m != 'verifyEmail' || o.isEmpty) return;

    verifying = true;
    verified = false;
    verifyError = null;
    notifyListeners();

    final st = await _service.applyActionCodeIfNeeded(
      mode: mode,
      oobCode: oobCode,
    );

    verifying = false;
    verified = st.verified;
    verifyError = st.error;
    notifyListeners();
  }

  // ------------------------------------------------------------
  // zip lookup (delegate)
  // ------------------------------------------------------------

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

    final res = await _service.lookupZip(zip7);
    if (res.ok) {
      prefCtrl.text = res.pref;
      cityCtrl.text = res.city;
      addr1Ctrl.text = res.town;
      zipError = null;
    } else {
      zipError = res.error;
    }

    zipLoading = false;
    notifyListeners();
  }

  // ------------------------------------------------------------
  // save (delegate)
  // ------------------------------------------------------------

  Future<void> saveAddressToBackend(BuildContext context) async {
    saving = true;
    saveMsg = null;
    notifyListeners();

    final res = await _service.saveAddress(
      lastName: lastNameCtrl.text,
      lastNameKana: lastNameKanaCtrl.text,
      firstName: firstNameCtrl.text,
      firstNameKana: firstNameKanaCtrl.text,
      zip: zipCtrl.text,
      pref: prefCtrl.text,
      city: cityCtrl.text,
      addr1: addr1Ctrl.text,
      addr2: addr2Ctrl.text,
    );

    saving = false;
    if (res.ok) {
      saveMsg = res.message;
      notifyListeners();
      if (!context.mounted) return;
      context.go('/billing-address');
      return;
    }

    saveMsg = res.error;
    notifyListeners();
  }

  void goSignIn(BuildContext context) {
    final fromEncoded = Uri.encodeComponent(
      GoRouterState.of(context).uri.toString(),
    );
    context.go('/login?from=$fromEncoded');
  }
}

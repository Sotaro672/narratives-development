// frontend\mall\lib\features\auth\application\billing_address_service.dart
import 'package:firebase_auth/firebase_auth.dart';

import '../../billingAddress/infrastructure/billing_address_repository_http.dart';

class BillingAddressSaveResult {
  BillingAddressSaveResult({
    required this.ok,
    required this.message,
    this.nextRoute,
  });

  final bool ok;
  final String message;
  final String? nextRoute;
}

/// BillingAddress の application service
/// - 入力正規化/バリデーション
/// - FirebaseAuth から uid を取得
/// - Repository を呼ぶ
/// - UI に必要な結果（message / nextRoute）を返す
class BillingAddressService {
  BillingAddressService({
    BillingAddressRepositoryHttp? repo,
    FirebaseAuth? auth,
    this.logger,
  }) : _repo = repo ?? BillingAddressRepositoryHttp(),
       _auth = auth ?? FirebaseAuth.instance;

  final BillingAddressRepositoryHttp _repo;
  final FirebaseAuth _auth;

  final void Function(String s)? logger;

  void dispose() {
    _repo.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  void _log(String s) => logger?.call(s);

  String backTo(String? from) {
    final f = _s(from);
    if (f.isNotEmpty) return f;
    return '/shipping-address';
  }

  // 全角数字を半角に寄せる（日本語IME対策）
  String _toAsciiDigits(String s) {
    return s.replaceAllMapped(RegExp(r'[０-９]'), (m) {
      final code = m[0]!.codeUnitAt(0);
      return String.fromCharCode(code - 0xFEE0);
    });
  }

  String normalizeDigits(String s) {
    final ascii = _toAsciiDigits(s);
    return ascii.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String normalizeCardNumber(String s) {
    // ハイフン/スペース/全角数字などを除去して数字だけに
    final ascii = _toAsciiDigits(s);
    return ascii.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String formatCardNumberForDisplay(String s) {
    final digits = normalizeCardNumber(s);
    final buf = StringBuffer();
    for (var i = 0; i < digits.length; i++) {
      if (i != 0 && i % 4 == 0) buf.write(' ');
      buf.write(digits[i]);
    }
    return buf.toString();
  }

  bool canSave({
    required bool saving,
    required String cardNumberRaw,
    required String holderRaw,
    required String cvcRaw,
  }) {
    if (saving) return false;

    final card = normalizeCardNumber(cardNumberRaw);
    final holder = _s(holderRaw);
    final cvc = normalizeDigits(cvcRaw);

    // 雛形: ざっくり必須チェック（Luhn等は後で）
    if (card.length < 12) return false; // AMEX等も考慮し ">=12" 程度に
    if (holder.isEmpty) return false;
    if (cvc.length != 3) return false;

    return true;
  }

  /// ✅ 実処理: uid を docId として PATCH /mall/billing-addresses/{uid}
  Future<BillingAddressSaveResult> save({
    required String cardNumberRaw,
    required String holderRaw,
    required String cvcRaw,
  }) async {
    try {
      final user = _auth.currentUser;
      if (user == null) {
        return BillingAddressSaveResult(ok: false, message: 'サインインが必要です。');
      }

      final uid = user.uid.trim();
      if (uid.isEmpty) {
        return BillingAddressSaveResult(ok: false, message: 'uid が取得できませんでした。');
      }

      final card = normalizeCardNumber(cardNumberRaw);
      final holder = _s(holderRaw);
      final cvc = normalizeDigits(cvcRaw);

      _log(
        'save start uid=$uid cardDigitsLen=${card.length} holder="$holder" cvcLen=${cvc.length}',
      );

      // ✅ POST ではなく PATCH（docId=uid 前提）
      await _repo.update(
        id: uid,
        cardNumber: card,
        cardholderName: holder,
        cvc: cvc,
      );

      _log('save ok uid=$uid');

      return BillingAddressSaveResult(
        ok: true,
        message: '請求情報を保存しました。',
        nextRoute: '/avatar-create',
      );
    } catch (e) {
      _log('save failed err=$e');
      return BillingAddressSaveResult(ok: false, message: e.toString());
    }
  }
}

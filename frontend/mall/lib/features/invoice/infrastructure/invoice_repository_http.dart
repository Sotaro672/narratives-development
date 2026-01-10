//frontend\mall\lib\features\invoice\infrastructure\invoice_repository_http.dart
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';

import '../../../app/config/api_base.dart';

class InvoiceRepositoryHttp {
  InvoiceRepositoryHttp({Dio? dio}) : _dio = dio ?? Dio();

  final Dio _dio;

  void dispose() {
    try {
      _dio.close(force: true);
    } catch (_) {
      // ignore
    }
  }

  Future<void> startCheckout({
    required String orderId,
    required String billingAddressId,
    required List<int> prices,
    int tax = 0,
    int shipping = 0,
  }) async {
    final token = await FirebaseAuth.instance.currentUser?.getIdToken();
    if (token == null || token.isEmpty) {
      throw Exception('auth token missing');
    }

    final oid = orderId.trim();
    if (oid.isEmpty) throw Exception('orderId is empty');

    final bid = billingAddressId.trim();
    if (bid.isEmpty) throw Exception('billingAddressId is empty');

    if (prices.isEmpty) throw Exception('prices is empty');

    final base = resolveApiBase();
    final url = '$base/mall/me/invoices';

    final body = <String, dynamic>{
      'orderId': oid,
      'billingAddressId': bid,
      'prices': prices,
      'tax': tax,
      'shipping': shipping,
    };

    final res = await _dio.post(
      url,
      options: Options(
        headers: {
          'Authorization': 'Bearer $token',
          'Content-Type': 'application/json',
        },
        // 4xx/5xx を例外にせずここで握る
        validateStatus: (code) => code != null,
      ),
      data: jsonEncode(body),
    );

    final sc = res.statusCode ?? 0;
    if (sc != 200 && sc != 201 && sc != 202 && sc != 204) {
      throw Exception('startCheckout failed status=$sc body=${res.data}');
    }
  }
}

import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';
// auth_token_provider is re-exported via core/core.dart

class ApiService extends GetxService {
  late Dio _dio;

  Future<void> init() async {
    final options = BaseOptions(
      baseUrl: apiUrl,
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
      // Timeouts generosos para procesamiento de PDFs grandes y operaciones IA
      // Aumentado a 200s para casos interactivos que pueden tardar hasta 180s en backend
      connectTimeout: const Duration(seconds: 30),
      receiveTimeout: const Duration(
        seconds: 200,
      ), // 200s: margen sobre backend 180s
      sendTimeout: const Duration(minutes: 2), // 120s: permite uploads grandes
    );
    _dio = Dio(options);
    _addTokenInterceptor(_dio);
  }

  Dio get dio => _dio;

  void _addTokenInterceptor(Dio dio) {
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (
          RequestOptions options,
          RequestInterceptorHandler handler,
        ) async {
          final token = await AuthTokenProvider.instance.getToken();
          if (token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }

          return handler.next(options);
        },
        onError: (DioException error, ErrorInterceptorHandler handler) async {
          final status = error.response?.statusCode;
          final uri = error.requestOptions.uri;
          dynamic payload = error.response?.data;
          // Normalize payload to string for clearer console logs
          String body;
          if (payload == null) {
            body = '<no-body>';
          } else if (payload is String) {
            body = payload;
          } else {
            try {
              body = payload.toString();
            } catch (_) {
              body = '<unprintable-body>';
            }
          }

          Logger.error(
            error.toString(),
            className: 'ApiService',
            methodName: 'dio.interceptors.add',
            meta: 'url: $uri | status: $status | body: $body',
          );
          // Retry once on 401 with the latest stored token.
          if (error.response?.statusCode == 401 &&
              error.requestOptions.extra['__retried__'] != true) {
            final latest = await AuthTokenProvider.instance.getToken();
            if (latest.isNotEmpty) {
              final req = error.requestOptions;
              req.headers['Authorization'] = 'Bearer $latest';
              req.extra = {...req.extra, '__retried__': true};
              try {
                final res = await dio.fetch(req);
                return handler.resolve(res);
              } catch (_) {}
            }
          }
          return handler.next(error);
        },
      ),
    );
  }
}

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
          Logger.error(
            error.toString(),
            className: 'ApiService',
            methodName: 'dio.interceptors.add',
            meta: 'url: ${error.requestOptions.uri}',
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

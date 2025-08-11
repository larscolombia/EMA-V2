import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:get/get.dart';


class ApiService extends GetxService {
  final _userService = Get.find<UserService>();
  final FlutterSecureStorage _storage = const FlutterSecureStorage();
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
        onRequest:
            (RequestOptions options, RequestInterceptorHandler handler) async {
          var token = _userService.currentUser.value.authToken;

          // If the token is not available in the user service, try to
          // retrieve it from secure storage to keep the session persistent.
          if (token.isEmpty) {
            token = await _storage.read(key: 'auth_token') ?? '';
          }

          if (token.isNotEmpty) {
            options.headers['Authorization'] = 'Bearer $token';
          }

          return handler.next(options);
        },
        onError: (DioException error, ErrorInterceptorHandler handler) {
          Logger.error(error.toString(), className: 'ApiService', methodName: 'dio.interceptors.add', meta: 'url: ${error.requestOptions.uri}');
          return handler.next(error);
        },
      ),
    );
  }
}

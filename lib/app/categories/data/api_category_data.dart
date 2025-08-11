import 'package:dio/dio.dart';
import 'package:ema_educacion_medica_avanzada/app/categories/models/category_model.dart';
import 'package:ema_educacion_medica_avanzada/core/api/api_service.dart';
import 'package:get/get.dart';


class ApiCategoryData extends GetxService {
  final Dio _dio = Get.find<ApiService>().dio;

  final List<CategoryModel> _categories = [];

  Future<List<CategoryModel>> getCategories() async {

    if (_categories.isNotEmpty) {
      return _categories;
    }

    final response = await _dio.get('/categoria-medicas');

    if (response.statusCode == 200) {
      final data = response.data;

      final downloaded = await data.map<CategoryModel>((c) => CategoryModel.fromApi(c)).toList();
      
      _categories.addAll(downloaded);

      return _categories;
    }
    throw Exception('Error al obtener las categorías');
  }
}

import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
import 'package:ema_educacion_medica_avanzada/app/categories/data/api_category_data.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:get/get.dart';



class CategoriesController extends GetxController {
  final _categoryData = Get.find<ApiCategoryData>();
  
  final Rx<CategoryModel> currentCategory = CategoryModel.empty().obs;
  
  // Observador para filtrar las categorías
  final categoryFilter = ''.obs;

  // Observador para actualizar CategoryTextFiel
  final categorySelectedName = ''.obs;

  // Observador de las categorías filtradas por CategoryTextField
  final categoriesFiltered = <CategoryModel>[].obs;

  final _categories = <CategoryModel>[].obs;

  @override
  void onInit() {
    super.onInit();

    _getCategories();

    debounce(
      categoryFilter,
      _filterCategories,
      time: Duration(milliseconds: 300)
    );
  }

  void _filterCategories(String namePart) {
    if (namePart.isEmpty) {
      categoriesFiltered.value = _categories;
      currentCategory.value = CategoryModel.empty();
      return;
    }

    final filteredCategories = _categories.where(
      (category) {
        final namePartLower = namePart.toLowerCase();
        return category.name.toLowerCase().contains(namePartLower);
      }
    ).toList();

    categoriesFiltered.value = filteredCategories;
  }

  Future<void> _getCategories() async {
    try {
      final data = await _categoryData.getCategories();

      _categories.addAll(data);
      _filterCategories('');

    } catch (e) {
      Notify.snackbar('Categorías', 'Error al cargar las categorías', NotifyType.error);
    }
  }

  void setCategoryFilter(String namePart) {
    categoryFilter.value = namePart;
  }
  
  void setCategorySelected(CategoryModel category) {
    categorySelectedName.value = category.name;
    categoryFilter.value = category.name;
    currentCategory.value = category;
  }
  
  void setCurrentCategory({required int categoryId}) {
    currentCategory.value = _categories.firstWhere(
      (element) => element.id == categoryId,
      orElse:() => CategoryModel.empty(),
    );
  }
}

class MostStudiedCategory {
  final int categoryId;
  final String categoryName;

  MostStudiedCategory({
    required this.categoryId,
    required this.categoryName,
  });

  factory MostStudiedCategory.fromJson(Map<String, dynamic> json) {
    return MostStudiedCategory(
      categoryId: json['category_id'],
      categoryName: json['category_name'],
    );
  }
}

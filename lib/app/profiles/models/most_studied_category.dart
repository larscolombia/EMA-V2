class MostStudiedCategory {
  final int categoryId;
  final String categoryName;
  final int studyCount;  // Cantidad de tests realizados en esta categoría

  MostStudiedCategory({
    required this.categoryId,
    required this.categoryName,
    required this.studyCount,
  });

  factory MostStudiedCategory.fromJson(Map<String, dynamic> json) {
    return MostStudiedCategory(
      categoryId: json['category_id'] as int? ?? 0,
      categoryName: json['category_name'] as String? ?? '',
      studyCount: json['study_count'] as int? ?? 0,
    );
  }

  Map<String, dynamic> toJson() => {
        'category_id': categoryId,
        'category_name': categoryName,
        'study_count': studyCount,
      };

  @override
  String toString() => toJson().toString();
}

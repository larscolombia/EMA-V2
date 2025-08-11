// ignore_for_file: public_member_api_docs, sort_constructors_first


class CategoryModel {
  final int id;
  final String name;
  final String? description;
  final DateTime createdAt;
  final DateTime updatedAt;

  CategoryModel({
    required this.id,
    required this.name,
    required this.createdAt,
    required this.updatedAt,
    this.description,
  });

  factory CategoryModel.empty() {
    return CategoryModel(
      id: 0,
      name: '',
      description: '',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );
  }

  // Todo: ajustar con el endpoint
  factory CategoryModel.fromApi(Map<String, dynamic> map) {
    return CategoryModel(
      id: map['id'] as int,
      name: map['name'] as String,
      description: map['description'] != null ? map['description'] as String : null,
      createdAt: DateTime.parse(map['created_at'] as String),
      updatedAt: DateTime.parse(map['updated_at'] as String),
    );
  }
}

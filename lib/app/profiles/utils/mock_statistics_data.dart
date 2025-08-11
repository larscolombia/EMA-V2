import 'package:flutter/material.dart';

class MockStatisticsData {
  // Datos para plan básico
  static const List<Map<String, dynamic>> basicStatistics = [
    {'label': 'Chats', 'count': 2},
    {'label': 'Cuestionarios', 'count': 1},
    {'label': 'Casos Clínicos', 'count': 1},
  ];

  static const Map<String, dynamic> basicPoints = {
    'earned': 50,
    'possible': 100,
  };

  static const List<int> basicMonthlyPoints = [
    20,
    30,
    50,
    40,
    25,
    35,
    45,
    30,
    20,
    15,
    25,
    30
  ];

  static const List<Map<String, dynamic>> basicCategories = [
    {
      'name': 'Medicina General',
      'icon': Icons.medical_services,
      'color': Colors.blue
    },
    {'name': 'Pediatría', 'icon': Icons.child_care, 'color': Colors.green},
    {'name': 'Medicina Interna', 'icon': Icons.healing, 'color': Colors.orange},
  ];
}

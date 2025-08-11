import 'package:flutter/material.dart';


class EmaOverlayRoute {
  String name;
  Widget Function()? topArea;
  Widget Function()? topList;
  Widget Function() bottomArea;

  EmaOverlayRoute({
    required this.name,
    required this.bottomArea,
    this.topArea,
    this.topList,
  });

  factory EmaOverlayRoute.empty() {
    return EmaOverlayRoute(
      name: '',
      topArea: null,
      topList: null,
      bottomArea: () => Center(child: Text('Ruta vacía')),
    );
  }
}

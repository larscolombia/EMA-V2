// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
// import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
// import 'package:ema_educacion_medica_avanzada/config/config.dart';
// import 'package:flutter/material.dart';
// import 'package:get/get.dart';


// class QuizzCategoryCard extends StatelessWidget {
//   final _categoriesController = Get.find<CategoriesController>();
//   final _quizzesController = Get.find<QuizzesController>();

//   QuizzCategoryCard({
//     super.key,
//   });

//   @override
//   Widget build(BuildContext context) {
//     return Obx(() {
//       // Empty category has the id 0
//       if (_categoriesController.currentCategory.value.id == 0) {
//         return const SizedBox();
//       }

//       final category = _categoriesController.currentCategory.value;

//       return Container(
//         decoration: BoxDecoration(
//           border: Border.all(color: AppStyles.tertiaryColor, width: 1),
//           borderRadius: BorderRadius.circular(18),
//         ),
//         margin: const EdgeInsets.only(top: 32),
//         padding: const EdgeInsets.all(18),
//         child: Column(
//           crossAxisAlignment: CrossAxisAlignment.stretch,
//           children: [
//             Text(
//               category.name,
//               style: AppStyles.subtitleBold,
//             ),
//             Text(
//               category.description ?? '',
//               style: AppStyles.subtitle,
//             ),
//             Row(
//               mainAxisAlignment: MainAxisAlignment.end,
//               children: [
//                 FilledButton(
//                   onPressed: () {
//                     _quizzesController.getByCategoryId(category.id);
//                     Get.offAllNamed(Routes.quizzesCategory.path(category.id.toString()));
//                   },
//                   style: ButtonStyle(
//                     backgroundColor: WidgetStateProperty.all(
//                       AppStyles.tertiaryColor,
//                     ),
//                   ),
//                   child: Text('Ver Categoría'),
//                 ),
//               ],
//             ),
//           ],
//         ),
//       );
//     });
//   }
// }

// import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
// import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
// import 'package:ema_educacion_medica_avanzada/config/config.dart';
// import 'package:flutter/material.dart';
// import 'package:get/get.dart';


// class QuizzesListByCategory extends GetView<QuizzesController> {
//   const QuizzesListByCategory({
//     super.key,
//   });

//   Widget onError(String? errorMessage) {
//     return Center(child: Text(errorMessage ?? 'Error al cargar quizzes'));
//   }

//   Widget onLoading() {
//     return const Center(child: CircularProgressIndicator());
//   }

//   Widget onEmpty() {
//     return const Center(child: Text('No hay quizzes disponibles'));
//   }

//   @override
//   Widget build(BuildContext context) {
//     return controller.obx(
//       (state) {
//         if (state == null || state.isEmpty) {
//           return const Center(child: Text('No hay quizzes disponibles'));
//         }

//         final quizzes = state.map((quizz) {
//           return TextButton.icon(
//             onPressed: () {
//               controller.setCurrent(quiz: quizz);
//               Get.toNamed(Routes.quizDetail.name, arguments: quizz);
//             },
//             style: TextButton.styleFrom(
//               padding: const EdgeInsets.all(8),
//               shape: RoundedRectangleBorder(
//                 borderRadius: BorderRadius.circular(8),
//               ),
//             ),
//             label: Text(
//               quizz.title,
//               softWrap: true,
//             ),
//             icon: AppIcons.arrowRightCircular(),
//             iconAlignment: IconAlignment.end,
//           );
//         }).toList();

//         return Container(
//           padding: const EdgeInsets.all(8),
//           decoration: BoxDecoration(
//             border: Border.all(
//               color: AppStyles.tertiaryColor,
//               width: 1,
//             ),
//             borderRadius: BorderRadius.circular(8),
//           ),
//           child: Column(
//             children: [...quizzes],
//           ),
//         );
//       },
//       onLoading: onLoading(),
//       onEmpty: onEmpty(),
//       onError: onError,
//     );
//   }
// }

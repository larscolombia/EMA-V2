// import 'package:ema_educacion_medica_avanzada/app/clinical_cases/clinical_cases.dart';
// import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
// import 'package:ema_educacion_medica_avanzada/config/config.dart';
// import 'package:flutter/material.dart';
// import 'package:get/get.dart';


// class ClilicalCasesListByHistory extends GetView<ClinicalCasesController> {
//   const ClilicalCasesListByHistory({
//     super.key,
//   });

//   Widget onLoading() {
//     return const Center(child: CircularProgressIndicator());
//   }

//   Widget onEmpty() {
//     return const Center(
//       child: Text('No tiene casos clínicos disponibles'),
//     );
//   }

//   Widget onError(String? errorMessage) {
//     return Center(
//       child: Text(errorMessage ?? 'Error al cargar los casos clínicos'),
//     );
//   }

//   @override
//   Widget build(BuildContext context) {
//     return controller.obx(
//       (state) {
//         if (state == null || state.isEmpty) {
//           return const Center(
//               child: Text('Aún no tiene casos clínicos disponibles'));
//         }

//         final clinicalCases = state.map((clinicalCase) {
//           return TextButton.icon(
//             onPressed: () {
//               controller.setClinicalCase(clinicalCase);
//               Get.toNamed(Routes.clinicalCaseDetail.path(clinicalCase.uid));
//             },
//             style: TextButton.styleFrom(
//               padding: const EdgeInsets.all(8),
//               shape: RoundedRectangleBorder(
//                 borderRadius: BorderRadius.circular(8),
//               ),
//             ),
//             label: Text(
//               clinicalCase.name,
//               softWrap: true,
//             ),
//             icon: AppIcons.arrowRightCircular(),
//             iconAlignment: IconAlignment.end,
//           );
//         }).toList();

//         return Column(
//           children: [
//             SizedBox(height: 32),

//             Align(
//               alignment: Alignment.center,
//               child: Text(
//                 'Tu Historial',
//                 style: AppStyles.title2,
//               ),
//             ),
            
//             SizedBox(height: 32),

//             Container(
//               padding: const EdgeInsets.all(8),
//               decoration: BoxDecoration(
//                 border: Border.all(
//                   color: AppStyles.tertiaryColor,
//                   width: 1,
//                 ),
//                 borderRadius: BorderRadius.circular(20),
//               ),
//               child: Column(
//                 children: [...clinicalCases],
//               ),
//             ),
//           ],
//         );
//       },
//       onLoading: onLoading(),
//       onEmpty: onEmpty(),
//       onError: onError,
//     );
//   }
// }

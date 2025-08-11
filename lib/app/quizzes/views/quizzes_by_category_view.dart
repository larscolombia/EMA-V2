// import 'package:ema_educacion_medica_avanzada/app/categories/categories.dart';
// import 'package:ema_educacion_medica_avanzada/app/quizzes/quizzes.dart';
// import 'package:ema_educacion_medica_avanzada/common/widgets.dart';
// import 'package:ema_educacion_medica_avanzada/config/config.dart';
// import 'package:ema_educacion_medica_avanzada/core/core.dart';
// import 'package:flutter/material.dart';
// import 'package:get/get.dart';


// class QuizzesByCategoryView extends StatelessWidget {
//   final _categoryController = Get.find<CategoriesController>();

//   QuizzesByCategoryView({super.key});

//   final bootomSheet = Container(
//     decoration: BoxDecoration(
//       color: AppStyles.whiteColor,
//     ),
//     child: Column(
//       mainAxisSize: MainAxisSize.min,
//       children: [
//         AppScaffold.footerCredits(),
//       ],
//     ),
//   );

//   void generateQuizz () {
//     // Todo: generateQuizz function
//     Logger.mini('generateQuizz()');
//   }

//   @override
//   Widget build(BuildContext context) {
//     return Scaffold(
//       appBar: AppScaffold.appBar(backRoute: Routes.quizzesHome.name),
//       extendBody: true,
//       body: SingleChildScrollView(
//         padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
//         child: Column(
//           // mainAxisAlignment: MainAxisAlignment.start,
//           mainAxisSize: MainAxisSize.max,
//           crossAxisAlignment: CrossAxisAlignment.stretch,
//           children: [
//             Text(
//               'Categorías > ${_categoryController.currentCategory.value.name}',
//               style: AppStyles.breadCrumb,
//             ),

//             SizedBox(height: 8),

//             Text(
//               _categoryController.currentCategory.value.description ?? '',
//               style: AppStyles.subtitle,
//             ),

//             SizedBox(height: 32),

//             QuizzesListByCategory(),

//             SizedBox(height: 32),

//             OutlineAiButton(text: 'Generame uno', onPressed: generateQuizz,),

//             SizedBox(height: 64),
//           ],
//         ),
//       ),
//       bottomNavigationBar: bootomSheet,
//     );
//   }
// }

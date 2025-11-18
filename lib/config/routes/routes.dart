part of 'app_pages.dart';

enum Routes {
  actionsList,
  clinicalCaseAnalytical,
  clinicalCaseInteractive,
  clinicalCaseEvaluation,
  // clinicalCasesHome,
  home,
  login,
  register,
  forgotPassword,
  resetPassword,
  profile,
  quizDetail,
  quizFeedBack,
  quizzesHome,
  quizzesCategory,
  start,
  subscriptions;

  String get name {
    switch (this) {
      case actionsList:
        return '/actions';

      case clinicalCaseAnalytical:
        return '/clinical-case-detail/:uid';

      case clinicalCaseInteractive:
        return '/clinical-case-interactive/:uid';
      case clinicalCaseEvaluation:
        return '/clinical-case-evaluation/:uid';

      // case clinicalCasesHome:
      //   return '/clinical-cases';

      case home:
        return '/';

      case login:
        return '/login';

      case register:
        return '/register';

      case forgotPassword:
        return '/forgot-password';

      case resetPassword:
        return '/reset-password';

      case profile:
        return '/profile/:uid';

      case subscriptions:
        return '/subscriptions';

      case quizDetail:
        return '/quiz/:uid';

      case quizFeedBack:
        return '/quiz-feedback/:uid';

      case quizzesHome:
        return '/quizzes';

      case quizzesCategory:
        return '/quizzes-category/:uid';

      case start:
        return '/start';
    }
  }

  String path([String uid = '']) {
    return uid.isEmpty ? name : '/${name.split('/')[1]}/$uid';
  }
}

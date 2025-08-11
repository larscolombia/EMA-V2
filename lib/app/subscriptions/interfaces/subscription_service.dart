import 'package:ema_educacion_medica_avanzada/app/subscriptions/subscriptions.dart';

abstract class SubscriptionService {
  Future<List<Subscription>> fetchSubscriptions({required String authToken});

  Future<void> updateSubscriptionQuantities({
    required int subscriptionId,
    required String authToken,
    int? consultations,
    int? questionnaires,
    int? clinicalCases,
    int? files,
  });
}

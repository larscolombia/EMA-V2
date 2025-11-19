class TimelineData {
  final List<TimelineDataPoint> points;
  final String period;
  final DateTime startDate;
  final DateTime endDate;

  TimelineData({
    required this.points,
    required this.period,
    required this.startDate,
    required this.endDate,
  });

  factory TimelineData.fromJson(
    Map<String, dynamic> json,
    String period,
    DateTime start,
    DateTime end,
  ) {
    return TimelineData(
      points:
          (json['data'] as List<dynamic>?)
              ?.map(
                (e) => TimelineDataPoint.fromJson(e as Map<String, dynamic>),
              )
              .toList() ??
          [],
      period: period,
      startDate: start,
      endDate: end,
    );
  }
}

class TimelineDataPoint {
  final String date;
  final int users;
  final double revenue;
  final int consultations;
  final int tests;
  final int clinicalCases;

  TimelineDataPoint({
    required this.date,
    required this.users,
    required this.revenue,
    required this.consultations,
    required this.tests,
    required this.clinicalCases,
  });

  factory TimelineDataPoint.fromJson(Map<String, dynamic> json) {
    return TimelineDataPoint(
      date: json['date'] ?? '',
      users: json['users'] ?? 0,
      revenue: (json['revenue'] ?? 0.0).toDouble(),
      consultations: json['consultations'] ?? 0,
      tests: json['tests'] ?? 0,
      clinicalCases: json['clinical_cases'] ?? 0,
    );
  }
}

class UserListItem {
  final int id;
  final String email;
  final String firstName;
  final String lastName;
  final DateTime createdAt;
  final String planName;
  final int subscriptionId;
  final bool hasSubscription;

  UserListItem({
    required this.id,
    required this.email,
    required this.firstName,
    required this.lastName,
    required this.createdAt,
    required this.planName,
    required this.subscriptionId,
    required this.hasSubscription,
  });

  String get fullName => '$firstName $lastName';

  factory UserListItem.fromJson(Map<String, dynamic> json) {
    return UserListItem(
      id: json['id'] ?? 0,
      email: json['email'] ?? '',
      firstName: json['first_name'] ?? '',
      lastName: json['last_name'] ?? '',
      createdAt: DateTime.tryParse(json['created_at'] ?? '') ?? DateTime.now(),
      planName: json['plan_name'] ?? '',
      subscriptionId: json['subscription_id'] ?? 0,
      hasSubscription: json['has_subscription'] ?? false,
    );
  }
}

class SubscriptionHistoryItem {
  final int id;
  final int userId;
  final String userEmail;
  final String userName;
  final int planId;
  final String planName;
  final double price;
  final DateTime startDate;
  final DateTime? endDate;
  final int frequency;
  final bool isActive;

  SubscriptionHistoryItem({
    required this.id,
    required this.userId,
    required this.userEmail,
    required this.userName,
    required this.planId,
    required this.planName,
    required this.price,
    required this.startDate,
    this.endDate,
    required this.frequency,
    required this.isActive,
  });

  factory SubscriptionHistoryItem.fromJson(Map<String, dynamic> json) {
    return SubscriptionHistoryItem(
      id: json['id'] ?? 0,
      userId: json['user_id'] ?? 0,
      userEmail: json['user_email'] ?? '',
      userName: json['user_name'] ?? '',
      planId: json['plan_id'] ?? 0,
      planName: json['plan_name'] ?? '',
      price: (json['price'] ?? 0.0).toDouble(),
      startDate: DateTime.tryParse(json['start_date'] ?? '') ?? DateTime.now(),
      endDate:
          json['end_date'] != null ? DateTime.tryParse(json['end_date']) : null,
      frequency: json['frequency'] ?? 0,
      isActive: json['is_active'] ?? false,
    );
  }
}

enum TimePeriod {
  day,
  week,
  month,
  year;

  String get value {
    switch (this) {
      case TimePeriod.day:
        return 'day';
      case TimePeriod.week:
        return 'week';
      case TimePeriod.month:
        return 'month';
      case TimePeriod.year:
        return 'year';
    }
  }

  String get label {
    switch (this) {
      case TimePeriod.day:
        return 'Día';
      case TimePeriod.week:
        return 'Semana';
      case TimePeriod.month:
        return 'Mes';
      case TimePeriod.year:
        return 'Año';
    }
  }

  int get defaultDays {
    switch (this) {
      case TimePeriod.day:
        return 30;
      case TimePeriod.week:
        return 90;
      case TimePeriod.month:
        return 180;
      case TimePeriod.year:
        return 730;
    }
  }
}

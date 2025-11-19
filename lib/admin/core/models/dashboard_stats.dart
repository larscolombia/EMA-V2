class DashboardStats {
  final UserStats users;
  final FinancialStats financial;
  final ActivityStats activity;
  final List<PlanStats> plans;
  final List<RecentActivityItem> recentActivity;

  DashboardStats({
    required this.users,
    required this.financial,
    required this.activity,
    required this.plans,
    required this.recentActivity,
  });

  factory DashboardStats.fromJson(Map<String, dynamic> json) {
    return DashboardStats(
      users: UserStats.fromJson(json['users'] ?? {}),
      financial: FinancialStats.fromJson(json['financial'] ?? {}),
      activity: ActivityStats.fromJson(json['activity'] ?? {}),
      plans:
          (json['plans'] as List<dynamic>?)
              ?.map((e) => PlanStats.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      recentActivity:
          (json['recent_activity'] as List<dynamic>?)
              ?.map(
                (e) => RecentActivityItem.fromJson(e as Map<String, dynamic>),
              )
              .toList() ??
          [],
    );
  }
}

class UserStats {
  final int total;
  final int active;
  final int newThisMonth;
  final double retentionRate;
  final double growthPercent;

  UserStats({
    required this.total,
    required this.active,
    required this.newThisMonth,
    required this.retentionRate,
    required this.growthPercent,
  });

  factory UserStats.fromJson(Map<String, dynamic> json) {
    return UserStats(
      total: json['total'] ?? 0,
      active: json['active'] ?? 0,
      newThisMonth: json['new_this_month'] ?? 0,
      retentionRate: (json['retention_rate'] ?? 0.0).toDouble(),
      growthPercent: (json['growth_percent'] ?? 0.0).toDouble(),
    );
  }
}

class FinancialStats {
  final double totalRevenue;
  final double monthlyRevenue;
  final double averageTicket;
  final double conversionRate;
  final double growthPercent;

  FinancialStats({
    required this.totalRevenue,
    required this.monthlyRevenue,
    required this.averageTicket,
    required this.conversionRate,
    required this.growthPercent,
  });

  factory FinancialStats.fromJson(Map<String, dynamic> json) {
    return FinancialStats(
      totalRevenue: (json['total_revenue'] ?? 0.0).toDouble(),
      monthlyRevenue: (json['monthly_revenue'] ?? 0.0).toDouble(),
      averageTicket: (json['average_ticket'] ?? 0.0).toDouble(),
      conversionRate: (json['conversion_rate'] ?? 0.0).toDouble(),
      growthPercent: (json['growth_percent'] ?? 0.0).toDouble(),
    );
  }
}

class ActivityStats {
  final int totalConsultations;
  final int totalTests;
  final int totalClinicalCases;

  ActivityStats({
    required this.totalConsultations,
    required this.totalTests,
    required this.totalClinicalCases,
  });

  factory ActivityStats.fromJson(Map<String, dynamic> json) {
    return ActivityStats(
      totalConsultations: json['total_consultations'] ?? 0,
      totalTests: json['total_tests'] ?? 0,
      totalClinicalCases: json['total_clinical_cases'] ?? 0,
    );
  }
}

class PlanStats {
  final int id;
  final String name;
  final int subscriberCount;
  final double percentage;
  final double revenue;

  PlanStats({
    required this.id,
    required this.name,
    required this.subscriberCount,
    required this.percentage,
    required this.revenue,
  });

  factory PlanStats.fromJson(Map<String, dynamic> json) {
    return PlanStats(
      id: json['id'] ?? 0,
      name: json['name'] ?? '',
      subscriberCount: json['subscriber_count'] ?? 0,
      percentage: (json['percentage'] ?? 0.0).toDouble(),
      revenue: (json['revenue'] ?? 0.0).toDouble(),
    );
  }
}

class RecentActivityItem {
  final String type;
  final String title;
  final String description;
  final DateTime timestamp;
  final String userEmail;

  RecentActivityItem({
    required this.type,
    required this.title,
    required this.description,
    required this.timestamp,
    required this.userEmail,
  });

  factory RecentActivityItem.fromJson(Map<String, dynamic> json) {
    return RecentActivityItem(
      type: json['type'] ?? '',
      title: json['title'] ?? '',
      description: json['description'] ?? '',
      timestamp: DateTime.tryParse(json['timestamp'] ?? '') ?? DateTime.now(),
      userEmail: json['user_email'] ?? '',
    );
  }
}

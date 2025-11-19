package stats

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	"ema-backend/login"
	"ema-backend/migrations"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

// Init sets the DB connection for stats queries
func Init(database *sql.DB) {
	db = database
}

// AdminStatsResponse represents the response structure for admin dashboard
type AdminStatsResponse struct {
	Users          UserStats            `json:"users"`
	Financial      FinancialStats       `json:"financial"`
	Activity       ActivityStats        `json:"activity"`
	Plans          []PlanStats          `json:"plans"`
	RecentActivity []RecentActivityItem `json:"recent_activity"`
}

type UserStats struct {
	Total         int     `json:"total"`
	Active        int     `json:"active"`
	NewThisMonth  int     `json:"new_this_month"`
	RetentionRate float64 `json:"retention_rate"`
	GrowthPercent float64 `json:"growth_percent"`
}

type FinancialStats struct {
	TotalRevenue   float64 `json:"total_revenue"`
	MonthlyRevenue float64 `json:"monthly_revenue"`
	AverageTicket  float64 `json:"average_ticket"`
	ConversionRate float64 `json:"conversion_rate"`
	GrowthPercent  float64 `json:"growth_percent"`
}

type ActivityStats struct {
	TotalConsultations int `json:"total_consultations"`
	TotalTests         int `json:"total_tests"`
	TotalClinicalCases int `json:"total_clinical_cases"`
}

type PlanStats struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	SubscriberCount int     `json:"subscriber_count"`
	Percentage      float64 `json:"percentage"`
	Revenue         float64 `json:"revenue"`
}

type RecentActivityItem struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	UserEmail   string    `json:"user_email"`
}

// RegisterAdminRoutes registers admin statistics endpoints
func RegisterAdminRoutes(r *gin.Engine) {
	r.GET("/admin/stats", requireSuperAdmin(), getAdminStats)
	r.GET("/admin/stats/timeline", requireSuperAdmin(), getTimeline)
	r.GET("/admin/stats/users/list", requireSuperAdmin(), getUsersList)
	r.GET("/admin/stats/subscriptions/history", requireSuperAdmin(), getSubscriptionsHistory)
}

// requireSuperAdmin middleware verifies the user is a super_admin
func requireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token requerido"})
			c.Abort()
			return
		}

		email, ok := login.GetEmailFromToken(token)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		user := migrations.GetUserByEmail(email)
		if user == nil || user.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acceso denegado: se requiere rol super_admin"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getAdminStats returns comprehensive statistics for the admin dashboard
func getAdminStats(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	log.Printf("[ADMIN_STATS] Fetching admin statistics")

	response := AdminStatsResponse{
		Users:          getUserStats(),
		Financial:      getFinancialStats(),
		Activity:       getActivityStats(),
		Plans:          getPlanStats(),
		RecentActivity: getRecentActivity(10),
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

func getUserStats() UserStats {
	stats := UserStats{}

	// Total users
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.Total)

	// Active users (with active subscriptions)
	db.QueryRow(`
		SELECT COUNT(DISTINCT s.user_id) 
		FROM subscriptions s 
		WHERE s.end_date IS NULL OR s.end_date > NOW()
	`).Scan(&stats.Active)

	// New users this month
	db.QueryRow(`
		SELECT COUNT(*) 
		FROM users 
		WHERE created_at >= DATE_FORMAT(NOW(), '%Y-%m-01')
	`).Scan(&stats.NewThisMonth)

	// New users last month for growth calculation
	var newLastMonth int
	db.QueryRow(`
		SELECT COUNT(*) 
		FROM users 
		WHERE created_at >= DATE_FORMAT(DATE_SUB(NOW(), INTERVAL 1 MONTH), '%Y-%m-01')
		  AND created_at < DATE_FORMAT(NOW(), '%Y-%m-01')
	`).Scan(&newLastMonth)

	// Calculate growth percentage
	if newLastMonth > 0 {
		stats.GrowthPercent = ((float64(stats.NewThisMonth) - float64(newLastMonth)) / float64(newLastMonth)) * 100
	} else if stats.NewThisMonth > 0 {
		stats.GrowthPercent = 100
	}

	// Retention rate (active / total)
	if stats.Total > 0 {
		stats.RetentionRate = (float64(stats.Active) / float64(stats.Total)) * 100
	}

	log.Printf("[ADMIN_STATS] Users: total=%d active=%d new_month=%d growth=%.2f%% retention=%.2f%%",
		stats.Total, stats.Active, stats.NewThisMonth, stats.GrowthPercent, stats.RetentionRate)

	return stats
}

func getFinancialStats() FinancialStats {
	stats := FinancialStats{}

	// Total revenue (sum of all paid subscriptions)
	db.QueryRow(`
		SELECT IFNULL(SUM(p.price), 0) 
		FROM subscriptions s 
		JOIN subscription_plans p ON s.plan_id = p.id 
		WHERE p.price > 0
	`).Scan(&stats.TotalRevenue)

	// Revenue this month
	db.QueryRow(`
		SELECT IFNULL(SUM(p.price), 0) 
		FROM subscriptions s 
		JOIN subscription_plans p ON s.plan_id = p.id 
		WHERE p.price > 0 
		  AND s.start_date >= DATE_FORMAT(NOW(), '%Y-%m-01')
	`).Scan(&stats.MonthlyRevenue)

	// Revenue last month for growth calculation
	var revenueLastMonth float64
	db.QueryRow(`
		SELECT IFNULL(SUM(p.price), 0) 
		FROM subscriptions s 
		JOIN subscription_plans p ON s.plan_id = p.id 
		WHERE p.price > 0 
		  AND s.start_date >= DATE_FORMAT(DATE_SUB(NOW(), INTERVAL 1 MONTH), '%Y-%m-01')
		  AND s.start_date < DATE_FORMAT(NOW(), '%Y-%m-01')
	`).Scan(&revenueLastMonth)

	// Calculate growth percentage
	if revenueLastMonth > 0 {
		stats.GrowthPercent = ((stats.MonthlyRevenue - revenueLastMonth) / revenueLastMonth) * 100
	} else if stats.MonthlyRevenue > 0 {
		stats.GrowthPercent = 100
	}

	// Average ticket (total revenue / paid subscriptions count)
	var paidSubsCount int
	db.QueryRow(`
		SELECT COUNT(*) 
		FROM subscriptions s 
		JOIN subscription_plans p ON s.plan_id = p.id 
		WHERE p.price > 0
	`).Scan(&paidSubsCount)

	if paidSubsCount > 0 {
		stats.AverageTicket = stats.TotalRevenue / float64(paidSubsCount)
	}

	// Conversion rate (paid subs / total users)
	var totalUsers int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if totalUsers > 0 {
		stats.ConversionRate = (float64(paidSubsCount) / float64(totalUsers)) * 100
	}

	log.Printf("[ADMIN_STATS] Financial: total_revenue=%.2f monthly=%.2f avg_ticket=%.2f conversion=%.2f%% growth=%.2f%%",
		stats.TotalRevenue, stats.MonthlyRevenue, stats.AverageTicket, stats.ConversionRate, stats.GrowthPercent)

	return stats
}

func getActivityStats() ActivityStats {
	stats := ActivityStats{}

	// Total consultations used (plan limits - remaining)
	db.QueryRow(`
		SELECT IFNULL(SUM(p.consultations - s.consultations), 0)
		FROM subscriptions s
		JOIN subscription_plans p ON s.plan_id = p.id
	`).Scan(&stats.TotalConsultations)

	// Total tests used
	db.QueryRow(`
		SELECT IFNULL(SUM(p.questionnaires - s.questionnaires), 0)
		FROM subscriptions s
		JOIN subscription_plans p ON s.plan_id = p.id
	`).Scan(&stats.TotalTests)

	// Total clinical cases used
	db.QueryRow(`
		SELECT IFNULL(SUM(p.clinical_cases - s.clinical_cases), 0)
		FROM subscriptions s
		JOIN subscription_plans p ON s.plan_id = p.id
	`).Scan(&stats.TotalClinicalCases)

	log.Printf("[ADMIN_STATS] Activity: consultations=%d tests=%d clinical_cases=%d",
		stats.TotalConsultations, stats.TotalTests, stats.TotalClinicalCases)

	return stats
}

func getPlanStats() []PlanStats {
	rows, err := db.Query(`
		SELECT 
			p.id,
			p.name,
			COUNT(s.id) as subscriber_count,
			IFNULL(SUM(p.price), 0) as revenue
		FROM subscription_plans p
		LEFT JOIN subscriptions s ON p.id = s.plan_id
		GROUP BY p.id, p.name
		ORDER BY subscriber_count DESC
	`)
	if err != nil {
		log.Printf("[ADMIN_STATS] Error fetching plan stats: %v", err)
		return []PlanStats{}
	}
	defer rows.Close()

	var plans []PlanStats
	var totalSubscribers int

	// First pass to get total subscribers
	for rows.Next() {
		var plan PlanStats
		rows.Scan(&plan.ID, &plan.Name, &plan.SubscriberCount, &plan.Revenue)
		totalSubscribers += plan.SubscriberCount
		plans = append(plans, plan)
	}

	// Calculate percentages
	for i := range plans {
		if totalSubscribers > 0 {
			plans[i].Percentage = (float64(plans[i].SubscriberCount) / float64(totalSubscribers)) * 100
		}
	}

	log.Printf("[ADMIN_STATS] Plans: total=%d total_subscribers=%d", len(plans), totalSubscribers)

	return plans
}

func getRecentActivity(limit int) []RecentActivityItem {
	rows, err := db.Query(`
		SELECT 
			'subscription' as type,
			u.email,
			p.name,
			s.start_date
		FROM subscriptions s
		JOIN users u ON s.user_id = u.id
		JOIN subscription_plans p ON s.plan_id = p.id
		ORDER BY s.start_date DESC
		LIMIT ?
	`, limit)
	if err != nil {
		log.Printf("[ADMIN_STATS] Error fetching recent activity: %v", err)
		return []RecentActivityItem{}
	}
	defer rows.Close()

	var activities []RecentActivityItem
	for rows.Next() {
		var activity RecentActivityItem
		var planName string
		rows.Scan(&activity.Type, &activity.UserEmail, &planName, &activity.Timestamp)

		activity.Title = "Nueva suscripción"
		activity.Description = "Usuario " + activity.UserEmail + " se suscribió a " + planName

		activities = append(activities, activity)
	}

	log.Printf("[ADMIN_STATS] Recent activity: count=%d", len(activities))

	return activities
}

// TimelineDataPoint represents a single point in a timeline chart
type TimelineDataPoint struct {
	Date          string  `json:"date"`
	Users         int     `json:"users"`
	Revenue       float64 `json:"revenue"`
	Consultations int     `json:"consultations"`
	Tests         int     `json:"tests"`
	ClinicalCases int     `json:"clinical_cases"`
}

// getTimeline returns timeline data for charts with period filter
// Query params: period (day|week|month|year), start_date, end_date
func getTimeline(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	period := c.DefaultQuery("period", "month") // day, week, month, year
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	var dateFormat string
	var intervalDays int

	switch period {
	case "day":
		dateFormat = "%Y-%m-%d"
		intervalDays = 30 // Last 30 days by default
	case "week":
		dateFormat = "%Y-%u" // Year-Week
		intervalDays = 90    // Last ~13 weeks
	case "year":
		dateFormat = "%Y"
		intervalDays = 730 // Last 2 years
	default: // month
		dateFormat = "%Y-%m"
		intervalDays = 180 // Last 6 months
	}

	// Set default date range if not provided
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -intervalDays).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	log.Printf("[TIMELINE] period=%s start=%s end=%s", period, startDate, endDate)

	// Query for new users per period
	usersQuery := `
		SELECT 
			period,
			COUNT(*) as count
		FROM (
			SELECT DATE_FORMAT(created_at, ?) as period
			FROM users
			WHERE created_at >= ? AND created_at <= ?
		) as user_periods
		GROUP BY period
		ORDER BY period ASC
	`

	rows, err := db.Query(usersQuery, dateFormat, startDate, endDate)
	if err != nil {
		log.Printf("[TIMELINE] Error fetching users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	usersByPeriod := make(map[string]int)
	for rows.Next() {
		var period string
		var count int
		rows.Scan(&period, &count)
		usersByPeriod[period] = count
	}

	// Query for revenue per period
	revenueQuery := `
		SELECT 
			period,
			IFNULL(SUM(price), 0) as revenue
		FROM (
			SELECT 
				DATE_FORMAT(s.start_date, ?) as period,
				p.price
			FROM subscriptions s
			JOIN subscription_plans p ON s.plan_id = p.id
			WHERE s.start_date >= ? AND s.start_date <= ? AND p.price > 0
		) as revenue_periods
		GROUP BY period
		ORDER BY period ASC
	`

	rows2, err := db.Query(revenueQuery, dateFormat, startDate, endDate)
	if err != nil {
		log.Printf("[TIMELINE] Error fetching revenue: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows2.Close()

	revenueByPeriod := make(map[string]float64)
	for rows2.Next() {
		var period string
		var revenue float64
		rows2.Scan(&period, &revenue)
		revenueByPeriod[period] = revenue
	}

	// Build timeline data points
	dataPoints := []TimelineDataPoint{}
	allPeriods := make(map[string]bool)

	for period := range usersByPeriod {
		allPeriods[period] = true
	}
	for period := range revenueByPeriod {
		allPeriods[period] = true
	}

	for period := range allPeriods {
		dataPoints = append(dataPoints, TimelineDataPoint{
			Date:    period,
			Users:   usersByPeriod[period],
			Revenue: revenueByPeriod[period],
			// Activity stats can be added here if needed
		})
	}

	log.Printf("[TIMELINE] Returning %d data points", len(dataPoints))
	c.JSON(http.StatusOK, gin.H{"data": dataPoints})
}

// UserListItem represents a user in the list view
type UserListItem struct {
	ID              int       `json:"id"`
	Email           string    `json:"email"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	CreatedAt       time.Time `json:"created_at"`
	PlanName        string    `json:"plan_name"`
	SubscriptionID  int       `json:"subscription_id"`
	HasSubscription bool      `json:"has_subscription"`
}

// getUsersList returns paginated list of users with filters
// Query params: limit, offset, search, has_subscription, plan_id, sort_by, order
func getUsersList(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	search := c.Query("search")
	hasSubscription := c.Query("has_subscription") // true, false, or empty (all)
	sortBy := c.DefaultQuery("sort_by", "created_at")
	order := c.DefaultQuery("order", "DESC")

	query := `
		SELECT 
			u.id,
			u.email,
			u.first_name,
			u.last_name,
			u.created_at,
			IFNULL(p.name, '') as plan_name,
			IFNULL(s.id, 0) as subscription_id,
			CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END as has_subscription
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id AND (s.end_date IS NULL OR s.end_date > NOW())
		LEFT JOIN subscription_plans p ON s.plan_id = p.id
		WHERE 1=1
	`

	args := []interface{}{}

	if search != "" {
		query += " AND (u.email LIKE ? OR u.first_name LIKE ? OR u.last_name LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	if hasSubscription == "true" {
		query += " AND s.id IS NOT NULL"
	} else if hasSubscription == "false" {
		query += " AND s.id IS NULL"
	}

	// Validate sort_by to prevent SQL injection
	allowedSorts := map[string]bool{
		"created_at": true,
		"email":      true,
		"first_name": true,
		"last_name":  true,
	}
	if !allowedSorts[sortBy] {
		sortBy = "created_at"
	}

	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	query += " ORDER BY u." + sortBy + " " + order + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("[USER_LIST] Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	users := []UserListItem{}
	for rows.Next() {
		var user UserListItem
		rows.Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.CreatedAt,
			&user.PlanName, &user.SubscriptionID, &user.HasSubscription)
		users = append(users, user)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM users u WHERE 1=1"
	countArgs := []interface{}{}
	if search != "" {
		countQuery += " AND (u.email LIKE ? OR u.first_name LIKE ? OR u.last_name LIKE ?)"
		searchPattern := "%" + search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}
	db.QueryRow(countQuery, countArgs...).Scan(&total)

	log.Printf("[USER_LIST] Returning %d users, total=%d", len(users), total)
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

// SubscriptionHistoryItem represents a subscription record
type SubscriptionHistoryItem struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	UserEmail string     `json:"user_email"`
	UserName  string     `json:"user_name"`
	PlanID    int        `json:"plan_id"`
	PlanName  string     `json:"plan_name"`
	Price     float64    `json:"price"`
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Frequency int        `json:"frequency"`
	IsActive  bool       `json:"is_active"`
}

// getSubscriptionsHistory returns paginated subscription history
// Query params: limit, offset, user_id, plan_id, active_only
func getSubscriptionsHistory(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")
	userID := c.Query("user_id")
	planID := c.Query("plan_id")
	activeOnly := c.Query("active_only") == "true"

	query := `
		SELECT 
			s.id,
			s.user_id,
			u.email,
			CONCAT(u.first_name, ' ', u.last_name) as user_name,
			s.plan_id,
			p.name as plan_name,
			p.price,
			s.start_date,
			s.end_date,
			s.frequency,
			CASE WHEN (s.end_date IS NULL OR s.end_date > NOW()) THEN 1 ELSE 0 END as is_active
		FROM subscriptions s
		JOIN users u ON s.user_id = u.id
		JOIN subscription_plans p ON s.plan_id = p.id
		WHERE 1=1
	`

	args := []interface{}{}

	if userID != "" {
		query += " AND s.user_id = ?"
		args = append(args, userID)
	}

	if planID != "" {
		query += " AND s.plan_id = ?"
		args = append(args, planID)
	}

	if activeOnly {
		query += " AND (s.end_date IS NULL OR s.end_date > NOW())"
	}

	query += " ORDER BY s.start_date DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("[SUBSCRIPTION_HISTORY] Error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	subscriptions := []SubscriptionHistoryItem{}
	for rows.Next() {
		var sub SubscriptionHistoryItem
		var endDate sql.NullTime
		rows.Scan(&sub.ID, &sub.UserID, &sub.UserEmail, &sub.UserName, &sub.PlanID,
			&sub.PlanName, &sub.Price, &sub.StartDate, &endDate, &sub.Frequency, &sub.IsActive)

		if endDate.Valid {
			sub.EndDate = &endDate.Time
		}

		subscriptions = append(subscriptions, sub)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM subscriptions s WHERE 1=1"
	countArgs := []interface{}{}
	if userID != "" {
		countQuery += " AND s.user_id = ?"
		countArgs = append(countArgs, userID)
	}
	if planID != "" {
		countQuery += " AND s.plan_id = ?"
		countArgs = append(countArgs, planID)
	}
	if activeOnly {
		countQuery += " AND (s.end_date IS NULL OR s.end_date > NOW())"
	}
	db.QueryRow(countQuery, countArgs...).Scan(&total)

	log.Printf("[SUBSCRIPTION_HISTORY] Returning %d subscriptions, total=%d", len(subscriptions), total)
	c.JSON(http.StatusOK, gin.H{"data": subscriptions, "total": total})
}
